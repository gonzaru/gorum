// by Gonzaru
// Distributed under the terms of the GNU General Public License v3

package gorum

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// local packages
import (
	"github.com/gonzaru/gorum/config"
	"github.com/gonzaru/gorum/utils"
)

// helpMenuFs shows fs' help menu information
func helpMenuFs() string {
	help := "help fs\n"
	help += ".                # lists the current directory contents [ls]\n"
	help += "-                # changes to parent directory [..,cd ..]\n"
	help += "cd               # changes to home user [cd ~]\n"
	help += "cd -             # changes to previous directory [_]\n"
	help += "cd /path/to/dir  # changes to directory\n"
	help += "clear            # clear the terminal screen\n"
	help += "exit             # exits the menu\n"
	help += "number           # plays the selected file\n"
	help += "pwd              # prints the current working directory\n"
	help += "stopplay         # stops playing the current media\n"
	help += "status           # prints status information\n"
	help += "mute             # toggles between mute and unmute\n"
	help += "pause            # toggles between pause and unpause\n"
	help += "video            # toggles between video auto and off\n"
	help += "help             # shows fs' help menu information [=]\n"
	return help
}

// menuFs plays selected media using a file selector
func menuFs() error {
	const maxOptErrors = 5
	var (
		numOptErrors int
		fileSel      string
		oldPwd       string
		selCur       string
		statusMsg    string
	)
	if !isRunning() {
		return errors.New(fmt.Sprintf("info: '%s' is not running, see help\n", config.ProgName))
	}
	homeDir, errUh := os.UserHomeDir()
	if errUh != nil {
		return errUh
	}
	curStream := streamPath()
	for {
		listFiles := make([]string, 0, 4096)
		cmdCc := exec.Command("clear")
		cmdCc.Stdout = os.Stdout
		if errCr := cmdCc.Run(); errCr != nil {
			return errCr
		}
		pwd, errOg := os.Getwd()
		if errOg != nil {
			return errOg
		}
		if statusMsg == "" {
			statusMsg = pwd
		}
		files, errRd := ioutil.ReadDir(pwd)
		if errRd != nil {
			return errRd
		}
		numPad := strconv.Itoa(utils.CountDigit(len(files)))
		fmt.Printf("%"+numPad+"s### %s ###\n", "", strings.ToUpper(config.ProgName))
		fmt.Printf("%"+numPad+"s=) help fs\n", "")
		pwdSplit := strings.Split(pwd, "/")
		parentDir := pwdSplit[len(pwdSplit)-2]
		curDir := pwdSplit[len(pwdSplit)-1]
		fmt.Printf("%"+numPad+"s-) ../ [%s]\n", "", parentDir)
		fmt.Printf("%"+numPad+"s.) ./ [%s]\n", "", curDir)
		for num, file := range files {
			listFiles = listFiles[0 : len(listFiles)+1]
			listFiles[num] = file.Name()
			filePath := fmt.Sprintf("%s/%s", pwd, file.Name())
			selCur = " "
			if fileSel == file.Name() || curStream == filePath {
				selCur = "*"
				if statusMsg == "" || statusMsg == pwd {
					statusMsg = file.Name()
				}
			}
			if file.IsDir() {
				fmt.Printf("%s%"+numPad+"d) %s/\n", selCur, num+1, file.Name())
			} else if file.Mode()&0111 != 0 && file.Mode()&os.ModeSymlink != os.ModeSymlink {
				fmt.Printf("%s%"+numPad+"d) %s*\n", selCur, num+1, file.Name())
			} else if file.Mode()&os.ModeNamedPipe != 0 {
				fmt.Printf("%s%"+numPad+"d) %s|\n", selCur, num+1, file.Name())
			} else if file.Mode()&os.ModeSocket != 0 {
				fmt.Printf("%s%"+numPad+"d) %s=\n", selCur, num+1, file.Name())
			} else if file.Mode()&os.ModeSymlink == os.ModeSymlink {
				fmt.Printf("%s%"+numPad+"d) %s@\n", selCur, num+1, file.Name())
			} else {
				fmt.Printf("%s%"+numPad+"d) %s\n", selCur, num+1, file.Name())
			}
		}
		fmt.Printf("\n# %s\n> ", strings.TrimRight(statusMsg, "\n"))
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		fileStr := strings.TrimSpace(scanner.Text())
		switch fileStr {
		case ".", "./", "ls":
			statusMsg = pwd
		case "-", "..", "../", "cd ..", "cd ../":
			if errCd := os.Chdir("../"); errCd != nil {
				statusMsg = errCd.Error()
				continue
			}
			oldPwd = pwd
			pwdParent, errGd := os.Getwd()
			if errGd != nil {
				statusMsg = errGd.Error()
				continue
			}
			statusMsg = pwdParent
		case "clear":
			statusMsg = ""
		case "cd", "cd ~", "cd ~/":
			if errCd := os.Chdir(homeDir); errCd != nil {
				statusMsg = errCd.Error()
				continue
			}
			oldPwd = pwd
			statusMsg = homeDir
		case "cd -", "_":
			if oldPwd != "" && oldPwd != pwd {
				if errCd := os.Chdir(oldPwd); errCd != nil {
					statusMsg = errCd.Error()
					continue
				}
				statusMsg = oldPwd
				oldPwd = pwd
			}
		case "exit":
			return nil
		case "=", "help", "help fs":
			statusMsg = helpMenuFs()
		case "mute", "pause", "video":
			if errTo := Toggle(fileStr); errTo != nil {
				statusMsg = errTo.Error()
				continue
			}
			cmd := fmt.Sprintf(`{"command": ["get_property_string", "%s"]}`, fileStr)
			_, content, errSc := sendCmd(cmd)
			if errSc != nil {
				log.Print(errSc)
				statusMsg = errSc.Error()
				continue
			}
			statusMsg = fmt.Sprintf("%s: %s", fileStr, content["data"])
		case "number":
			statusMsg = fmt.Sprintf("info: simply put the file %s and press ENTER", fileStr)
		case "pwd":
			statusMsg = pwd
		case "status":
			content, err := Status()
			if err != nil {
				statusMsg = err.Error()
			} else {
				statusMsg = "status\n" + content
			}
		case "stopplay":
			curStream = ""
			fileSel = ""
			statusMsg = ""
			if err := PlayStop(); err != nil {
				statusMsg = err.Error()
			}
		default:
			if fileId, errSa := strconv.Atoi(fileStr); errSa == nil {
				if fileId < 1 || fileId > len(listFiles) {
					numOptErrors++
					if numOptErrors >= maxOptErrors {
						return errors.New("menuFs: error: too many consecutive errors\n")
					}
					statusMsg = "invalid option"
					continue
				}
				fileSel = listFiles[fileId-1]
			} else {
				fileSel = fileStr
			}
			regexDir := regexp.MustCompile(`^cd\s`)
			regexHome := regexp.MustCompile(`^cd ~/`)
			fileDir := regexDir.ReplaceAllString(regexHome.ReplaceAllString(fileSel, homeDir+"/"), "")
			if fi, errOs := os.Stat(fileDir); errOs == nil && fi.IsDir() {
				if errCd := os.Chdir(fileDir); errCd != nil {
					statusMsg = errCd.Error()
					continue
				}
				oldPwd = pwd
				newPwd, errGw := os.Getwd()
				if errGw != nil {
					statusMsg = errGw.Error()
					continue
				}
				statusMsg = newPwd
				continue
			}
			match := false
			for _, file := range listFiles {
				if file == fileSel {
					match = true
					break
				}
			}
			if !match {
				numOptErrors++
				if numOptErrors >= maxOptErrors {
					return errors.New("menuFs: error: too many consecutive errors\n")
				}
				statusMsg = "invalid option"
				continue
			}
			fileAbs, errFa := filepath.Abs(fileSel)
			if errFa != nil {
				statusMsg = errFa.Error()
				continue
			}
			if errPl := Play(fileAbs); errPl != nil {
				statusMsg = errPl.Error()
				continue
			}
			curStream = ""
			statusMsg = fileSel
			cmd := `{"command": ["get_property", "filtered-metadata"]}`
			if _, errSc := StatusCmd(cmd, "error"); errSc != nil {
				statusMsg = errSc.Error()
			}
		}
	}
}
