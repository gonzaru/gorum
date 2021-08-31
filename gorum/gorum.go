// by Gonzaru
// Distributed under the terms of the GNU General Public License v3

package gorum

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// local packages
import (
	"github.com/gonzaru/gorum/config"
	"github.com/gonzaru/gorum/utils"
)

// checkOS checks if current operating system has been tested
func checkOS() bool {
	status := false
	items := []string{"freebsd", "linux", "netbsd", "openbsd"}
	for _, item := range items {
		if item == runtime.GOOS {
			status = true
			break
		}
	}
	return status
}

// CheckOut checks for a valid setup
func CheckOut() error {
	if !checkOS() {
		wrnMsg := fmt.Sprintf("checkOut: warning: '%s' has not been tested\n", runtime.GOOS)
		utils.ErrPrint(wrnMsg)
		log.Printf(wrnMsg)
		time.Sleep(time.Second)
	}
	cmds := []string{"clear", "reset", "stty", config.Player}
	for _, cmd := range cmds {
		if _, errLp := exec.LookPath(cmd); errLp != nil {
			return errors.New(fmt.Sprintf("checkOut: error: command '%s' not found\n", cmd))
		}
	}
	return nil
}

// cleanUp removes temporary files if necessary
func cleanUp() error {
	files := []string{
		config.LockDir,
		config.PidFile,
		config.PlayerControlFile,
		config.PlayerPidFile,
		config.WmFile,
	}
	for _, file := range files {
		if _, errOs := os.Stat(file); errOs == nil {
			if errOr := os.Remove(file); errOr != nil {
				return errOr
			}
		}
	}
	if errWb := wmBarUpdate(); errWb != nil {
		return errWb
	}
	return nil
}

// controlFileExists checks if controlFile exists and is in socket mode
func controlFileExists(file string) (bool, error) {
	fi, err := os.Stat(config.PlayerControlFile)
	if err != nil {
		return false, errors.New(fmt.Sprintf("controlFileExists: error: '%s' no such file or directory\n", file))
	}
	if fi.Mode()&os.ModeSocket == 0 {
		return false, errors.New(fmt.Sprintf("controlFileExists: error: '%s' is not a socket file\n", file))
	}
	return true, nil
}

// finish performs actions before leaving the program
func finish() error {
	if errSp := stopPlayer(); errSp != nil {
		return errSp
	}
	if errCu := cleanUp(); errCu != nil {
		return errCu
	}
	return nil
}

// Help shows help information
func Help() {
	fmt.Printf("Usage:\n")
	fmt.Printf("  %s number         # number key id from config.Streams\n", config.ProgName)
	fmt.Printf("  %s url            # plays the stream url\n", config.ProgName)
	fmt.Printf("  %s /path/to/file  # plays the local file\n", config.ProgName)
	fmt.Printf("  %s start          # starts %s\n", config.ProgName, config.ProgName)
	fmt.Printf("  %s stop           # stops %s\n", config.ProgName, config.ProgName)
	fmt.Printf("  %s stopplay       # stops playing the current media file [stopp]\n", config.ProgName)
	fmt.Printf("  %s status         # prints status information\n", config.ProgName)
	fmt.Printf("  %s mute           # toggles between mute and unmute\n", config.ProgName)
	fmt.Printf("  %s pause          # toggles between pause and unpause\n", config.ProgName)
	fmt.Printf("  %s video          # toggles between video auto and off\n", config.ProgName)
	fmt.Printf("  %s menu           # opens an interactive menu\n", config.ProgName)
	fmt.Printf("  %s help           # shows help menu information\n", config.ProgName)
}

// isIdle checks if no file is loaded
func isIdle() bool {
	status := false
	cmd := `{"command": ["get_property_string", "idle-active"]}`
	if _, content, errSc := sendCmd(cmd); errSc == nil && content["data"] == "yes" {
		status = true
	}
	return status
}

// isRunning checks if ProgName is locked or already running
func isRunning() bool {
	if fi, errOs := os.Stat(config.LockDir); errOs == nil && fi.IsDir() {
		return true
	}
	if utils.PidFileExists(config.PidFile) {
		return true
	}
	return false
}

// Play plays media files
func Play(file string) error {
	if !isRunning() {
		return errors.New(fmt.Sprintf("play: error: '%s' is not running\n", config.ProgName))
	}
	if os.Getenv("DISPLAY") == "" {
		cmd := `{"command": ["set_property", "video", false]}`
		if _, _, errSc := sendCmd(cmd); errSc != nil {
			return errSc
		}
	}
	streamInt, errSa := strconv.Atoi(file)
	if errSa == nil {
		if errPs := playStream(streamInt); errPs != nil {
			return errPs
		}
	} else {
		if errPf := playFile(file); errPf != nil {
			return errPf
		}
	}
	if errWb := wmBarUpdate(); errWb != nil {
		return errWb
	}
	return nil
}

// playFile plays streaming media files or local files
func playFile(file string) error {
	var (
		fileLoad string
		isLocal  = false
		isStream = false
	)
	if fi, errOs := os.Stat(file); errOs == nil {
		if fi.IsDir() {
			return errors.New(fmt.Sprintf("playFile: error: '%s' is a directory, not a file\n", file))
		}
		isLocal = true
	} else if utils.ValidUrl(file) {
		isStream = true
		fileLoad = file
	}
	if !isLocal && !isStream {
		return errors.New(fmt.Sprintf("playFile: error: '%s' no such file or stream url\n", file))
	}
	if isLocal {
		fileAbs, errFa := filepath.Abs(file)
		if errFa != nil {
			return errFa
		}
		fileLoad = fileAbs
	}
	if errPs := PlayStop(); errPs != nil {
		return errPs
	}
	if _, errOs := os.Stat(config.WmFile); errOs == nil {
		if errOr := os.Remove(config.WmFile); errOr != nil {
			return errOr
		}
	}
	cmd := "{\"command\": [\"loadfile\", \"" + fileLoad + "\", \"replace\"]}"
	if _, _, errSc := sendCmd(cmd); errSc != nil {
		return errSc
	}
	return nil
}

// playStream plays streaming media files
func playStream(stream int) error {
	streams := config.Streams
	if _, ok := streams[stream]["url"]; !ok {
		return errors.New(fmt.Sprintf("playStream: error: key map '%d' not found in streams\n", stream))
	}
	if errPs := PlayStop(); errPs != nil {
		return errPs
	}
	if _, errOs := os.Stat(config.WmFile); errOs == nil {
		if errOr := os.Remove(config.WmFile); errOr != nil {
			return errOr
		}
	}
	cmd := "{\"command\": [\"loadfile\", \"" + streams[stream]["url"] + "\", \"replace\"]}"
	if _, _, errSc := sendCmd(cmd); errSc != nil {
		return errSc
	}
	return nil
}

// PlayStop stops playing the current media
func PlayStop() error {
	if !isRunning() {
		return errors.New(fmt.Sprintf("playStop: error: '%s' is not running\n", config.ProgName))
	}
	if isIdle() {
		return nil
	}
	cmds := []string{
		`{"command": ["playlist-remove", "current"]}`,
		`{"command": ["stop"]}`,
	}
	if _, errSc := sendCmds(cmds, false); errSc != nil {
		return errSc
	}
	return nil
}

// sendCmd sends command to media player
func sendCmd(cmd string) ([]byte, map[string]interface{}, error) {
	var content map[string]interface{}
	if !isRunning() {
		return nil, nil, errors.New(fmt.Sprintf("sendCmd: error: '%s' is not running\n", config.ProgName))
	}
	if !json.Valid([]byte(cmd)) {
		return nil, nil, errors.New(fmt.Sprintf("sendCmd: error: invalid json %s\n", cmd))
	}
	if status, errCf := controlFileExists(config.PlayerControlFile); !status || errCf != nil {
		return nil, nil, errCf
	}
	conn, errNd := net.Dial("unix", config.PlayerControlFile)
	if errNd != nil {
		return nil, nil, errNd
	}
	defer func() {
		if errCc := conn.Close(); errCc != nil {
			utils.ErrPrint(errCc)
			log.Fatal(errCc)
		}
	}()
	sendData := cmd + "\n"
	if num, errCw := conn.Write([]byte(sendData)); num != len(sendData) || errCw != nil {
		return nil, nil, errCw
	}
	// avoids [ipc_0] Write error (Broken pipe)
	time.Sleep(time.Millisecond * 100)
	recvData := make([]byte, 1024)
	if _, errCr := conn.Read(recvData); errCr != nil {
		return nil, nil, errCr
	}
	dataJson := bytes.Split(recvData, []byte{'\n'})[0]
	if !json.Valid(dataJson) {
		return nil, nil, errors.New(fmt.Sprintf("sendCmd: error: invalid json %s\n", cmd))
	}
	if errJu := json.Unmarshal(dataJson, &content); errJu != nil {
		return nil, nil, errJu
	}
	return dataJson, content, nil
}

// sendCmds sends commands to media player
func sendCmds(cmds []string, async bool) ([][]interface{}, error) {
	var (
		dataJson []byte
		content  map[string]interface{}
		err      error
	)
	if !isRunning() {
		return nil, errors.New(fmt.Sprintf("sendCmds: error: '%s' is not running\n", config.ProgName))
	}
	arrSc := make([][]interface{}, len(cmds))
	if async {
		var wg sync.WaitGroup
		wg.Add(len(cmds))
		for num, cmd := range cmds {
			go func(num int, cmd string) {
				defer wg.Done()
				dataJson, content, err = sendCmd(cmd)
				arrSc[num] = append(arrSc[num], dataJson)
				arrSc[num] = append(arrSc[num], content)
				arrSc[num] = append(arrSc[num], err)
				if err != nil {
					utils.ErrPrint(err)
					log.Fatal(err)
				}
			}(num, cmd)
		}
		wg.Wait()
	} else {
		for num, cmd := range cmds {
			dataJson, content, err = sendCmd(cmd)
			arrSc[num] = append(arrSc[num], dataJson)
			arrSc[num] = append(arrSc[num], content)
			arrSc[num] = append(arrSc[num], err)
			if err != nil {
				return arrSc, err
			}
		}
	}
	return arrSc, nil
}

// SetLog sets logging output file
func SetLog() error {
	// create file if does not exist or append it
	file, err := os.OpenFile(config.GorumLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	log.SetOutput(file)
	return nil
}

// setUp
func setUp() error {
	if errCd := os.Mkdir(config.LockDir, 0700); errCd != nil {
		return errCd
	}
	if errCp := ioutil.WriteFile(config.PidFile, []byte(strconv.Itoa(os.Getpid())+"\n"), 0600); errCp != nil {
		return errCp
	}
	return nil
}

// SignalHandler sets signal handler events
func SignalHandler() {
	chSignal := make(chan os.Signal, 1)
	chExit := make(chan int)
	signal.Notify(chSignal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	go func() {
		for {
			sig := <-chSignal
			msg := fmt.Sprintf("signalHandler: info: recived signal '%s'\n", sig)
			fmt.Print(msg)
			log.Print(msg)
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				if err := finish(); err != nil {
					log.Fatal(err)
				}
				chExit <- 0
			case syscall.SIGHUP:
				// signal hang up
			default:
				errMsg := fmt.Sprintf("signalHandler: error: unsupported signal '%s'\n", sig)
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

// Start starts the ProgName
func Start() error {
	if isRunning() {
		return errors.New(fmt.Sprintf("start: error: '%s' is already running or locked\n", config.ProgName))
	}
	if errSu := setUp(); errSu != nil {
		return errSu
	}
	defer func() {
		if errFi := finish(); errFi != nil {
			utils.ErrPrint(errFi)
			log.Fatal(errFi)
		}
	}()
	msg := fmt.Sprintf("start: info: starting '%s'\n", config.ProgName)
	log.Print(msg)
	cmd := exec.Command(config.Player, config.PlayerArgs...)
	stdout, errSp := cmd.StdoutPipe()
	if errSp != nil {
		return errSp
	}
	cmd.Stderr = cmd.Stdout
	if errCs := cmd.Start(); errCs != nil {
		return errCs
	}
	if errCp := ioutil.WriteFile(config.PlayerPidFile, []byte(strconv.Itoa(cmd.Process.Pid)+"\n"), 0600); errCp != nil {
		return errCp
	}
	msg = fmt.Sprintf(
		"start: info: %s pid: %d, %s pid: %d\n", config.ProgName, os.Getpid(), config.Player, cmd.Process.Pid,
	)
	fmt.Print(msg)
	log.Print(msg)
	log.Printf("start: info: run %s\n", strings.Join(cmd.Args, " "))
	title := ""
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		regexTitle := regexp.MustCompile(`^\sicy-title:\s|^Title:\s|^\[cplayer].*/force-media-title=`)
		if strings.Contains(line, "[file] Opening ") {
			lineSplit := strings.Split(line, "/")
			title = strings.TrimSpace(lineSplit[len(lineSplit)-1])
		} else if match := regexTitle.Find([]byte(line)); len(match) > 0 {
			title = strings.Trim(strings.TrimRight(regexTitle.ReplaceAllString(line, ""), " -> 1"), `"`)
		} else {
			continue
		}
		if title != "" {
			log.Printf("start: title: %s\n", title)
			if errWf := wmFileUpdate(config.WmFile, []byte(title+"\n"), config.WmFilePerms); errWf != nil {
				return errWf
			}
		}
	}
	if errCw := cmd.Wait(); errCw != nil {
		return errors.New(fmt.Sprintf("start: error: '%s' command error %s\n", config.Player, errCw.Error()))
	}
	return nil
}

// Status prints status information
func Status() (string, error) {
	if !isRunning() {
		return "", errors.New(fmt.Sprintf("status: error: '%s' is not running\n", config.ProgName))
	}
	content, err := statusPlayer()
	return content, err
}

// StatusCmd checks command status
func StatusCmd(cmd string, field string, maxTries int) (map[string]interface{}, error) {
	var (
		content map[string]interface{}
		err     error = nil
		errSc   error = nil
	)
	if !isRunning() {
		return nil, errors.New(fmt.Sprintf("statusCmd: error: '%s' is not running\n", config.ProgName))
	}
	for i := 0; i < maxTries; i++ {
		time.Sleep(time.Second)
		_, content, errSc = sendCmd(cmd)
		if errSc != nil {
			err = errSc
			break
		}
		if _, okFi := content[field]; okFi && content["error"] == "success" {
			err = nil
			break
		}
		if _, okDa := content["data"]; okDa {
			if _, okFi := content["data"].(map[string]interface{})[field]; okFi && content["error"] == "success" {
				break
			}
		}
		if i == maxTries-1 {
			err = errors.New(fmt.Sprintf("statusCmd: error: property '%s' unavailable\n", field))
		}
	}
	return content, err
}

// statusPlayer prints media player status information
func statusPlayer() (string, error) {
	cmd := `{"command": ["get_property", "metadata"]}`
	dataJson, _, errSc := sendCmd(cmd)
	if errSc != nil {
		return "", errSc
	}
	outPretty, errJp := utils.JsonPretty(dataJson, "", "    ")
	if errJp != nil {
		return "", errJp
	}
	cmds := []string{
		`{"command": ["get_property_string", "mute"]}`,
		`{"command": ["get_property_string", "pause"]}`,
		`{"command": ["get_property_string", "video"]}`,
		`{"command": ["get_property_string", "idle-active"]}`,
		`{"command": ["get_property_string", "media-title"]}`,
		`{"command": ["get_property_string", "path"]}`,
		`{"command": ["get_property_string", "file-format"]}`,
	}
	arrSc, errSm := sendCmds(cmds, false)
	if errSm != nil {
		return "", errSm
	}
	statusInfo := fmt.Sprintf(
		"mute:  %s\npause: %s\nvideo: %s\nidle:  %s\nsong:  %v\npath:  %v\nffmt:  %v\nmeta:\n%s\n",
		arrSc[0][1].(map[string]interface{})["data"],
		arrSc[1][1].(map[string]interface{})["data"],
		arrSc[2][1].(map[string]interface{})["data"],
		arrSc[3][1].(map[string]interface{})["data"],
		arrSc[4][1].(map[string]interface{})["data"],
		arrSc[5][1].(map[string]interface{})["data"],
		arrSc[6][1].(map[string]interface{})["data"],
		outPretty,
	)
	return statusInfo, nil
}

// streamPath returns the active stream path
func streamPath() string {
	if !isRunning() {
		return ""
	}
	cmd := `{"command": ["get_property_string", "path"]}`
	_, content, errSc := sendCmd(cmd)
	if errSc != nil {
		return ""
	}
	return fmt.Sprintf("%v", content["data"])
}

// Stop stops the ProgName
func Stop() error {
	if !isRunning() {
		return errors.New(fmt.Sprintf("stop: error: '%s' is not running\n", config.ProgName))
	}
	defer func() {
		if errFi := finish(); errFi != nil {
			utils.ErrPrint(errFi)
			log.Fatal(errFi)
		}
	}()
	log.Printf("stop: info: stopping '%s'\n", config.ProgName)
	cmds := []string{
		`{"command": ["playlist-remove", "current"]}`,
		`{"command": ["stop"]}`,
		`{"command": ["quit"]}`,
	}
	if _, errSc := sendCmds(cmds, false); errSc != nil {
		return errSc
	}
	return nil
}

// stopPlayer stops the media player
func stopPlayer() error {
	if !utils.PidFileExists(config.PlayerPidFile) {
		return nil
	}
	content, errRf := ioutil.ReadFile(config.PlayerPidFile)
	if errRf != nil {
		return errRf
	}
	pid, errSa := strconv.Atoi(strings.TrimRight(string(content), "\n"))
	if errSa != nil {
		return errSa
	}
	if errSk := syscall.Kill(pid, syscall.SIGINT); errSk != nil {
		return errSk
	}
	return nil
}

// Toggle toggles property option
func Toggle(property string) error {
	if !isRunning() {
		return errors.New(fmt.Sprintf("toggle: error: '%s' is not running\n", config.ProgName))
	}
	if property == "video" {
		if errTv := toggleVideo(); errTv != nil {
			return errTv
		}
	} else {
		cmd := fmt.Sprintf(`{"command": ["cycle", "%s"]}`, property)
		if _, _, errSc := sendCmd(cmd); errSc != nil {
			return errSc
		}
	}
	return nil
}

// toggleVideo toggles between video auto and off
func toggleVideo() error {
	cmdGv := `{"command": ["get_property_string", "video"]}`
	_, content, errGv := sendCmd(cmdGv)
	if errGv != nil {
		return errGv
	}
	if content["data"] == "auto" || content["data"] == "yes" || content["data"] == "1" {
		cmdSf := `{"command": ["set_property", "video", false]}`
		if _, _, errSf := sendCmd(cmdSf); errSf != nil {
			return errSf
		}
	} else {
		cmdSa := `{"command": ["set_property", "video", "auto"]}`
		if _, _, errSa := sendCmd(cmdSa); errSa != nil {
			return errSa
		}
	}
	return nil
}

// wmBarUpdate updates window manager status bar
func wmBarUpdate() error {
	if config.WmDoBarUpdate {
		if _, errLp := exec.LookPath("wmbarupdate"); errLp == nil {
			cmd := exec.Command("wmbarupdate")
			if errCr := cmd.Run(); errCr != nil {
				return errCr
			}
		}
	}
	return nil
}

// wmFileUpdate updates window manager song title file
func wmFileUpdate(file string, data []byte, fi os.FileMode) error {
	var err error = nil
	if errWf := ioutil.WriteFile(file, data, fi); errWf != nil {
		err = errWf
	}
	if errWb := wmBarUpdate(); errWb != nil {
		err = errWb
	}
	return err
}
