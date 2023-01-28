// by Gonzaru
// Distributed under the terms of the GNU General Public License v3

package menu

import (
	"bufio"
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

// menuFile data type
type menuFile struct {
	numErrors int
	progTitle string
	statusMsg string
	streams   map[int]map[string]string
}

// finishMenu performs actions before leaving the menu
func finishMenu() error {
	var fileFlag string
	switch runtime.GOOS {
	case "linux":
		fileFlag = "-F"
	default:
		fileFlag = "-f"
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

// help shows help menu information
func (mf *menuFile) help() string {
	var help strings.Builder
	minVolStr := strconv.Itoa(config.VolumeMin)
	maxVolStr := strconv.Itoa(config.VolumeMax)
	help.WriteString("help\n")
	help.WriteString("clear       # clear the terminal screen\n")
	help.WriteString("exit        # exits the menu [quit]\n")
	help.WriteString("sf          # launches sf selector file [.]\n")
	help.WriteString("number      # plays the selected media stream\n")
	help.WriteString("url         # plays the stream url\n")
	help.WriteString("start       # starts " + mf.progTitle + "\n")
	help.WriteString("stop        # stops " + mf.progTitle + "\n")
	help.WriteString("stopplay    # stops playing the current media [stopp]\n")
	help.WriteString("status      # prints status information\n")
	help.WriteString("seek +n/-n  # seeks forward (+n) or backward (-n) number in seconds\n")
	help.WriteString("title       # prints media title\n")
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

// streamIds get the numeric stream ids
func (mf *menuFile) streamIds() []int {
	keys := make([]int, 0, len(mf.streams))
	for key := range mf.streams {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	return keys
}

// draw draw the menu
func (mf *menuFile) draw() error {
	if errSc := screen.Clear(); errSc != nil {
		return errSc
	}
	var selStream string
	curStream := gorum.StreamPath()
	numPad := strconv.Itoa(utils.CountDigit(len(mf.streams)))
	fmt.Printf("%"+numPad+"s### %s ###\n", "", strings.ToUpper(mf.progTitle))
	fmt.Printf("%"+numPad+"s?) help\n", "")
	fmt.Printf("%"+numPad+"s.) sf\n", "")
	for _, key := range mf.streamIds() {
		selStream = " "
		if curStream == mf.streams[key]["url"] {
			selStream = "*"
			if mf.statusMsg == "" {
				mf.statusMsg = mf.streams[key]["name"]
			}
		}
		fmt.Printf("%s%"+numPad+"d) %s\n", selStream, key, mf.streams[key]["name"])
	}
	fmt.Printf("\n# %s\n> ", strings.TrimRight(mf.statusMsg, "\n"))
	return nil
}

// doActionDefault executes the default menu option
func (mf *menuFile) doActionDefault(action string) error {
	var (
		errSa    error
		streamId int
	)
	streamId, errSa = strconv.Atoi(action)
	if _, ok := mf.streams[streamId]; (!ok || errSa != nil) && !utils.ValidUrl(action) {
		mf.numErrors++
		if mf.numErrors >= config.MaxMenuTries {
			errMsg := fmt.Errorf("doActionDefault: error: too many consecutive errors\n")
			utils.ErrPrint(errMsg)
			log.Fatal(errMsg)
		}
		return fmt.Errorf("error: invalid option")
	}
	mf.numErrors = 0
	if errPl := gorum.Play(action); errPl != nil {
		return errPl
	}
	cmd := `{"command": ["get_property", "filtered-metadata"]}`
	if _, errSc := gorum.StatusCmd(cmd, "error", config.MaxStatusTries); errSc != nil {
		return errSc
	}
	if _, ok := mf.streams[streamId]; ok {
		mf.statusMsg = mf.streams[streamId]["name"]
	} else {
		for key := range mf.streams {
			if action == mf.streams[key]["url"] {
				mf.statusMsg = mf.streams[key]["name"]
				break
			}
		}
	}
	if mf.statusMsg == "" {
		mf.statusMsg = action
	}
	return nil
}

// doActionSeek executes the seek menu option
func (mf *menuFile) doActionSeek(action string, actionArgs []string) error {
	regexSeek := regexp.MustCompile(`^seek\s[+-]\d+$`)
	if len(actionArgs) == 0 || !regexSeek.MatchString(action+" "+actionArgs[0]) {
		return fmt.Errorf("doActionSeek: error: invalid arg")
	}
	secondsInt, errSa := strconv.Atoi(actionArgs[0])
	if errSa != nil {
		return errSa
	}
	if errSe := gorum.Seek(secondsInt); errSe != nil {
		return errSe
	}
	cmd := `{"command": ["get_property_string", "playback-time"]}`
	_, content, errSc := gorum.SendCmd(cmd)
	if errSc != nil {
		return errSc
	}
	mf.statusMsg = fmt.Sprintf("%s: %s", "time", content["data"])
	return nil
}

// doActionStart executes the start menu option
func (mf *menuFile) doActionStart() error {
	if gorum.IsRunning() {
		return fmt.Errorf("doActionStart: error: '%s' is already running\n", mf.progTitle)
	}
	curFile, errOe := os.Executable()
	if errOe != nil {
		return errOe
	}
	cmdCg := exec.Command(curFile, "start")
	cmdCg.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if errCr := cmdCg.Start(); errCr != nil {
		return errCr
	}
	mf.statusMsg = fmt.Sprintf("info: %s pid: %s", mf.progTitle, strconv.Itoa(cmdCg.Process.Pid))
	return nil
}

// doActionToggle executes the toggle menu option
func (mf *menuFile) doActionToggle(action string) error {
	if errTo := gorum.Toggle(action); errTo != nil {
		return errTo
	}
	cmd := fmt.Sprintf(`{"command": ["get_property_string", "%s"]}`, action)
	_, content, errSc := gorum.SendCmd(cmd)
	if errSc != nil {
		return errSc
	}
	mf.statusMsg = fmt.Sprintf("%s: %s", action, content["data"])
	return nil
}

// doActionVolume executes the volume menu option
func (mf *menuFile) doActionVolume(actionBase string, actionArg []string) error {
	regexVolume := regexp.MustCompile(`^(volume|vol)\s-?\d+$`)
	if len(actionArg) == 0 || !regexVolume.MatchString(actionBase+" "+actionArg[0]) {
		return fmt.Errorf("doActionVolume: error: invalid arg")
	}
	numInt, errSa := strconv.Atoi(actionArg[0])
	if errSa != nil {
		return errSa
	}
	if errSe := gorum.Volume(numInt); errSe != nil {
		return errSe
	}
	mf.statusMsg = fmt.Sprintf("%s: %d", "volume", numInt)
	return nil
}

// doAction executes the selected menu option
func (mf *menuFile) doAction(option string) error {
	action := strings.Split(option, " ")[0]
	actionArgs := strings.Split(option, " ")[1:]
	mf.statusMsg = ""
	switch action {
	case ".", "sf":
		if err := sf.Run(); err != nil {
			mf.statusMsg = err.Error()
		}
	case "?", "help":
		mf.statusMsg = mf.help()
	case "clear":
		mf.statusMsg = ""
	case "exit", "quit":
		os.Exit(0)
	case "mute", "pause", "video":
		if err := mf.doActionToggle(action); err != nil {
			mf.statusMsg = err.Error()
		}
	case "number", "url":
		mf.statusMsg = fmt.Sprintf("info: simply put the stream %s and press ENTER", action)
	case "seek":
		if err := mf.doActionSeek(action, actionArgs); err != nil {
			mf.statusMsg = err.Error()
		}
	case "start":
		if err := mf.doActionStart(); err != nil {
			mf.statusMsg = err.Error()
		}
	case "status":
		content, err := gorum.Status()
		if err != nil {
			mf.statusMsg = err.Error()
		} else {
			mf.statusMsg = "status\n" + content
		}
	case "stop":
		if err := gorum.Stop(); err != nil {
			mf.statusMsg = err.Error()
		}
		if !gorum.IsRunning() {
			mf.statusMsg = fmt.Sprintf("info: '%s' is not running, see help\n", mf.progTitle)
		}
	case "stopp", "stopplay":
		if err := gorum.PlayStop(); err != nil {
			mf.statusMsg = err.Error()
		}
	case "title":
		content, err := gorum.Title()
		if err != nil {
			mf.statusMsg = err.Error()
		} else {
			mf.statusMsg = fmt.Sprintf("%s: %s", action, content)
		}
	case "volume", "vol":
		if err := mf.doActionVolume(action, actionArgs); err != nil {
			mf.statusMsg = err.Error()
		}
	default:
		if err := mf.doActionDefault(action); err != nil {
			mf.statusMsg = err.Error()
		}
	}
	return nil
}

// Menu plays the selected media using a streaming selector
func Menu() error {
	mf := menuFile{
		progTitle: config.ProgName,
		streams:   config.Streams,
	}
	if !gorum.IsRunning() {
		mf.statusMsg = fmt.Sprintf("info: '%s' is not running, see help\n", mf.progTitle)
	}
	for {
		if errDr := mf.draw(); errDr != nil {
			return errDr
		}
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		option := strings.TrimSpace(scanner.Text())
		if option == "exit" || option == "quit" {
			break
		}
		if errDo := mf.doAction(option); errDo != nil {
			mf.statusMsg = errDo.Error()
		}
	}
	return nil
}
