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
	"regexp"
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
		return errEc
	}
	cmdSs := exec.Command("stty", "sane")
	cmdSs.Stdin = os.Stdin
	if errCr := cmdSs.Run(); errCr != nil {
		return errCr
	}
	return nil
}

// helpMenu shows help menu information
func helpMenu() string {
	var help strings.Builder
	progName := config.ProgName
	minVolStr := strconv.Itoa(config.VolumeMin)
	maxVolStr := strconv.Itoa(config.VolumeMax)
	help.WriteString("help\n")
	help.WriteString("clear       # clear the terminal screen\n")
	help.WriteString("exit        # exits the menu\n")
	help.WriteString("sf          # launches sf selector file [.]\n")
	help.WriteString("number      # plays the selected media stream\n")
	help.WriteString("url         # plays the stream url\n")
	help.WriteString("start       # starts " + progName + "\n")
	help.WriteString("stop        # stops " + progName + "\n")
	help.WriteString("stopplay    # stops playing the current media [stopp]\n")
	help.WriteString("status      # prints status information\n")
	help.WriteString("seek +n/-n  # seeks forward (+n) or backward (-n) number in seconds\n")
	help.WriteString("mute        # toggles between mute and unmute\n")
	help.WriteString("pause       # toggles between pause and unpause\n")
	help.WriteString("video       # toggles between video auto and off\n")
	help.WriteString("volume n    # sets volume number between (" + minVolStr + "-" + maxVolStr + ") [vol]\n")
	help.WriteString("help        # shows help menu information [?]\n")
	return help.String()
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
		optionStr    string
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
		optionStr = strings.TrimSpace(scanner.Text())
		// options with arguments
		regexSeek := regexp.MustCompile(`^seek\s[+-]\d+$`)
		regexVolume := regexp.MustCompile(`^volume\s[-]?\d+$`)
		switch {
		case regexSeek.MatchString(optionStr):
			secondsInt, errSa := strconv.Atoi(strings.Split(optionStr, " ")[1])
			if errSa != nil {
				statusMsg = errSa.Error()
				continue
			}
			if errSe := gorum.Seek(secondsInt); errSe != nil {
				statusMsg = errSe.Error()
				continue
			}
			cmd := `{"command": ["get_property_string", "playback-time"]}`
			_, content, errSc := gorum.SendCmd(cmd)
			if errSc != nil {
				statusMsg = errSc.Error()
				continue
			}
			statusMsg = fmt.Sprintf("%s: %s", "time", content["data"])
			continue
		case regexVolume.MatchString(optionStr):
			numInt, errSa := strconv.Atoi(strings.Split(optionStr, " ")[1])
			if errSa != nil {
				statusMsg = errSa.Error()
				continue
			}
			if errSe := gorum.Volume(numInt); errSe != nil {
				statusMsg = errSe.Error()
				continue
			}
			statusMsg = fmt.Sprintf("%s: %d", "volume", numInt)
			continue
		}
		// options without arguments
		switch optionStr {
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
			if errTo := gorum.Toggle(optionStr); errTo != nil {
				statusMsg = errTo.Error()
				continue
			}
			cmd := fmt.Sprintf(`{"command": ["get_property_string", "%s"]}`, optionStr)
			_, content, errSc := gorum.SendCmd(cmd)
			if errSc != nil {
				statusMsg = errSc.Error()
				continue
			}
			statusMsg = fmt.Sprintf("%s: %s", optionStr, content["data"])
		case "number", "url":
			statusMsg = fmt.Sprintf("info: simply put the stream %s and press ENTER", optionStr)
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
			if errCr := cmdCg.Start(); errCr != nil {
				statusMsg = errCr.Error()
				continue
			}
			statusMsg = fmt.Sprintf("info: %s pid: %s", config.ProgName, strconv.Itoa(cmdCg.Process.Pid))
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
			streamId, errSa = strconv.Atoi(optionStr)
			if _, ok := streams[streamId]; (!ok || errSa != nil) && !utils.ValidUrl(optionStr) {
				numOptErrors++
				if numOptErrors >= maxOptErrors {
					return errors.New("menu: error: too many consecutive errors\n")
				}
				statusMsg = "invalid option"
				continue
			}
			numOptErrors = 0
			if errPl := gorum.Play(optionStr); errPl != nil {
				statusMsg = errPl.Error()
				continue
			}
			cmd := `{"command": ["get_property", "filtered-metadata"]}`
			if _, errSc := gorum.StatusCmd(cmd, "error", 5); errSc != nil {
				statusMsg = errSc.Error()
				continue
			}
			if _, ok := streams[streamId]; ok {
				statusMsg = streams[streamId]["name"]
			} else {
				for key := range streams {
					if optionStr == streams[key]["url"] {
						statusMsg = streams[key]["name"]
						streamId = key
						break
					}
				}
			}
			if statusMsg == "" {
				statusMsg = optionStr
			}
		}
	}
}
