// by Gonzaru
// Distributed under the terms of the GNU General Public License v3

package menu

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

// local packages
import (
	"github.com/gonzaru/gorum/config"
	"github.com/gonzaru/gorum/gorum"
	"github.com/gonzaru/gorum/screen"
	"github.com/gonzaru/gorum/sf"
	"github.com/gonzaru/gorum/utils"
)

// finishMenu performs actions before leaving the menu
func finishMenu() error {
	fileFlag := "-f"
	if runtime.GOOS == "linux" {
		fileFlag = "-F"
	}
	if errEc := exec.Command("stty", fileFlag, "/dev/tty", "echo").Run(); errEc != nil {
		log.Fatal(errEc)
	}
	cmdSs := exec.Command("stty", "sane")
	cmdSs.Stdin = os.Stdin
	if errCr := cmdSs.Run(); errCr != nil {
		log.Fatal(errCr)
	}
	return nil
}

// helpMenu shows help menu information
func helpMenu() string {
	help := "help\n"
	help += "clear     # clear the terminal screen\n"
	help += "exit      # exits the menu\n"
	help += "sf        # launches sf selector file [.]\n"
	help += "number    # plays the selected media stream\n"
	help += "url       # plays the stream url\n"
	help += "start     # starts " + config.ProgName + "\n"
	help += "stop      # stops " + config.ProgName + "\n"
	help += "stopplay  # stops playing the current media [stopp]\n"
	help += "status    # prints status information\n"
	help += "mute      # toggles between mute and unmute\n"
	help += "pause     # toggles between pause and unpause\n"
	help += "video     # toggles between video auto and off\n"
	help += "help      # shows help menu information [?]\n"
	return help
}

// SignalHandler sets signal handler
func SignalHandler() {
	chSignal := make(chan os.Signal, 1)
	chExit := make(chan int)
	signal.Notify(chSignal, syscall.SIGINT)
	go func() {
		for {
			sig := <-chSignal
			msg := fmt.Sprintf("\nsignalHandler: info: recived signal '%s'\n", sig)
			fmt.Print(msg)
			log.Print(msg)
			switch sig {
			case syscall.SIGINT:
				if err := finishMenu(); err != nil {
					utils.ErrPrint(err)
					log.Fatal(err)
				}
				chExit <- 0
			default:
				errMsg := fmt.Sprintf("\nsignalHandler: error: unsupported signal '%s'\n", sig)
				utils.ErrPrint(errMsg)
				log.Print(errMsg)
				if err := finishMenu(); err != nil {
					utils.ErrPrint(err)
					log.Fatal(err)
				}
				chExit <- 1
			}
		}
	}()
	code := <-chExit
	os.Exit(code)
}

// Menu plays selected media using a streaming selector
func Menu() error {
	const maxOptErrors = 5
	var (
		numOptErrors int
		streamId     int
		selCur       string
		statusMsg    string
		streamStr    string
	)
	streams := config.Streams
	keys := make([]int, 0, len(config.Streams))
	for key := range streams {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	if !gorum.IsRunning() {
		statusMsg = fmt.Sprintf("info: '%s' is not running, see help\n", config.ProgName)
	} else if streamId > 0 {
		if _, ok := streams[streamId]; ok {
			statusMsg = streams[streamId]["name"]
		}
	}
	curStream := gorum.StreamPath()
	numPad := strconv.Itoa(utils.CountDigit(len(streams)))
	for {
		if errSc := screen.Clear(); errSc != nil {
			return errSc
		}
		fmt.Printf("%"+numPad+"s### %s ###\n", "", strings.ToUpper(config.ProgName))
		fmt.Printf("%"+numPad+"s?) help\n", "")
		fmt.Printf("%"+numPad+"s.) sf\n", "")
		for _, key := range keys {
			selCur = " "
			if streamId == key || curStream == streams[key]["url"] {
				selCur = "*"
				if statusMsg == "" {
					statusMsg = streams[key]["name"]
				}
			}
			fmt.Printf("%s%"+numPad+"d) %s\n", selCur, key, streams[key]["name"])
		}
		fmt.Printf("\n# %s\n> ", strings.TrimRight(statusMsg, "\n"))
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		streamStr = strings.TrimSpace(scanner.Text())
		switch streamStr {
		case ".", "sf":
			if errMe := sf.Run(); errMe != nil {
				statusMsg = errMe.Error()
			}
		case "?", "help":
			statusMsg = helpMenu()
		case "clear":
			statusMsg = ""
		case "exit":
			return nil
		case "mute", "pause", "video":
			if errTo := gorum.Toggle(streamStr); errTo != nil {
				statusMsg = errTo.Error()
				continue
			}
			cmd := fmt.Sprintf(`{"command": ["get_property_string", "%s"]}`, streamStr)
			_, content, errSc := gorum.SendCmd(cmd)
			if errSc != nil {
				log.Print(errSc)
				statusMsg = errSc.Error()
				continue
			}
			statusMsg = fmt.Sprintf("%s: %s", streamStr, content["data"])
		case "number", "url":
			statusMsg = fmt.Sprintf("info: simply put the stream %s and press ENTER", streamStr)
		case "start":
			statusMsg = ""
			if gorum.IsRunning() {
				statusMsg = fmt.Sprintf("menu: error: '%s' is already running\n", config.ProgName)
				continue
			}
			curFile, errOe := os.Executable()
			if errOe != nil {
				statusMsg = errOe.Error()
				continue
			}
			cmdCg := exec.Command(curFile, "start")
			cmdCg.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			stdout, errSp := cmdCg.StdoutPipe()
			if errSp != nil {
				statusMsg = errSp.Error()
				continue
			}
			cmdCg.Stderr = cmdCg.Stdout
			if errCr := cmdCg.Start(); errCr != nil {
				statusMsg = errCr.Error()
				continue
			}
			buf := make([]byte, 1024)
			if _, errSr := stdout.Read(buf); errSr != nil {
				statusMsg = errSr.Error()
				continue
			}
			statusMsg = strings.Replace(string(buf), "\n", "", -1)
			if errSc := stdout.Close(); errSc != nil {
				statusMsg = errSc.Error()
			}
		case "status":
			content, err := gorum.Status()
			if err != nil {
				statusMsg = err.Error()
			} else {
				statusMsg = "status\n" + content
			}
		case "stop":
			curStream = ""
			streamId = -1
			statusMsg = ""
			if err := gorum.Stop(); err != nil {
				statusMsg = err.Error()
			}
		case "stopp", "stopplay":
			curStream = ""
			streamId = -1
			statusMsg = ""
			if err := gorum.PlayStop(); err != nil {
				statusMsg = err.Error()
			}
		default:
			var errSa error
			curStream = ""
			statusMsg = ""
			streamId, errSa = strconv.Atoi(streamStr)
			if _, ok := streams[streamId]; (!ok || errSa != nil) && !utils.ValidUrl(streamStr) {
				numOptErrors++
				if numOptErrors >= maxOptErrors {
					return errors.New("menu: error: too many consecutive errors\n")
				}
				statusMsg = "invalid option"
				continue
			}
			numOptErrors = 0
			if errPl := gorum.Play(streamStr); errPl != nil {
				log.Print(errPl)
				statusMsg = errPl.Error()
				continue
			}
			cmd := `{"command": ["get_property", "filtered-metadata"]}`
			if _, errSc := gorum.StatusCmd(cmd, "error", 5); errSc != nil {
				log.Print(errSc)
				statusMsg = errSc.Error()
				fmt.Println(statusMsg)
				continue
			}
			if _, ok := streams[streamId]; ok {
				statusMsg = streams[streamId]["name"]
			} else {
				for key := range streams {
					if streamStr == streams[key]["url"] {
						statusMsg = streams[key]["name"]
						streamId = key
						break
					}
				}
			}
			if statusMsg == "" {
				statusMsg = streamStr
			}
		}
	}
}
