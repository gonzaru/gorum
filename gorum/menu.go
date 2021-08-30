// by Gonzaru
// Distributed under the terms of the GNU General Public License v3

package gorum

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

// local packages
import (
	"github.com/gonzaru/gorum/config"
	"github.com/gonzaru/gorum/screen"
	"github.com/gonzaru/gorum/utils"
)

// finishMenu performs actions before leaving the menu
func finishMenu() error {
	if err := exec.Command("stty", "-F", "/dev/tty", "echo").Run(); err != nil {
		log.Fatal(err)
	}
	if err := exec.Command("stty", "sane").Run(); err != nil {
		log.Fatal(err)
	}
	if err := exec.Command("reset").Run(); err != nil {
		log.Fatal(err)
	}
	return nil
}

// helpMenu shows help menu information
func helpMenu() string {
	help := "help\n"
	help += "clear          # clear the terminal screen\n"
	help += "exit           # exits the menu\n"
	help += "fs             # launches the fs menu [.]\n"
	help += "number         # plays the selected media stream\n"
	help += "url            # plays the stream url\n"
	help += fmt.Sprintf("start          # starts %s\n", config.ProgName)
	help += fmt.Sprintf("stop           # stops %s\n", config.ProgName)
	help += "stopplay       # stops playing the current media [stopp]\n"
	help += "status         # prints status information\n"
	help += "mute           # toggles between mute and unmute\n"
	help += "pause          # toggles between pause and unpause\n"
	help += "video          # toggles between video auto and off\n"
	help += "help           # shows help menu information [=]\n"
	return help
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
	if !isRunning() {
		statusMsg = fmt.Sprintf("info: '%s' is not running, see help\n", config.ProgName)
	} else if streamId > 0 {
		if _, ok := streams[streamId]; ok {
			statusMsg = streams[streamId]["name"]
		}
	}
	curStream := streamPath()
	numPad := strconv.Itoa(utils.CountDigit(len(streams)))
	for {
		if errSc := screen.Clear(); errSc != nil {
			return errSc
		}
		fmt.Printf("%"+numPad+"s### %s ###\n", "", strings.ToUpper(config.ProgName))
		fmt.Printf("%"+numPad+"s=) help\n", "")
		fmt.Printf("%"+numPad+"s.) fs\n", "")
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
		case ".", "fs":
			if errMe := menuFs(); errMe != nil {
				statusMsg = errMe.Error()
			}
		case "=", "help":
			statusMsg = helpMenu()
		case "clear":
			statusMsg = ""
		case "exit":
			return nil
		case "mute", "pause", "video":
			if errTo := Toggle(streamStr); errTo != nil {
				statusMsg = errTo.Error()
				continue
			}
			cmd := fmt.Sprintf(`{"command": ["get_property_string", "%s"]}`, streamStr)
			_, content, errSc := sendCmd(cmd)
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
			if isRunning() {
				statusMsg = fmt.Sprintf("menu: error: '%s' is already running\n", config.ProgName)
				continue
			}
			curFile, errOe := os.Executable()
			if errOe != nil {
				statusMsg = errOe.Error()
				continue
			}
			cmdCg := exec.Command("setsid", curFile, "start")
			cmdCg.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
			if errCr := cmdCg.Run(); errCr != nil {
				statusMsg = errCr.Error()
			}
		case "status":
			content, err := Status()
			if err != nil {
				statusMsg = err.Error()
			} else {
				statusMsg = "status\n" + content
			}
		case "stop":
			curStream = ""
			streamId = -1
			statusMsg = ""
			if err := Stop(); err != nil {
				statusMsg = err.Error()
			}
		case "stopp", "stopplay":
			curStream = ""
			streamId = -1
			statusMsg = ""
			if err := PlayStop(); err != nil {
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
			if errPl := Play(streamStr); errPl != nil {
				log.Print(errPl)
				statusMsg = errPl.Error()
				continue
			}
			cmd := `{"command": ["get_property", "filtered-metadata"]}`
			if _, errSc := StatusCmd(cmd, "error", 5); errSc != nil {
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

// SignalHandlerMenu sets signal handler for menu
func SignalHandlerMenu() {
	chSignal := make(chan os.Signal, 1)
	chExit := make(chan int)
	signal.Notify(chSignal, syscall.SIGINT)
	go func() {
		for {
			sig := <-chSignal
			msg := fmt.Sprintf("\nsignalHandlerMenu: info: recived signal '%s'\n", sig)
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
				errMsg := fmt.Sprintf("\nsignalHandlerMenu: error: unsupported signal '%s'\n", sig)
				utils.ErrPrint(errMsg)
				log.Print(errMsg)
				if err := finish(); err != nil {
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
