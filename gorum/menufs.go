// by Gonzaru
// Distributed under the terms of the GNU General Public License v3

package gorum

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"math"
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
	"github.com/gonzaru/gorum/cursor"
	"github.com/gonzaru/gorum/screen"
	"github.com/gonzaru/gorum/utils"
)

// drawSelHeader draws selfs header
func drawSelHeader(pad string) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	pwdSplit := strings.Split(pwd, "/")
	parentDir := pwdSplit[len(pwdSplit)-2]
	curDir := pwdSplit[len(pwdSplit)-1]
	fmt.Printf("%"+pad+"s### %s ###\n", "", strings.ToUpper(config.ProgName))
	fmt.Printf("%"+pad+"s=) help (j,k,J,K,Enter,Escape,-,.)\n", "")
	fmt.Printf("%"+pad+"s-) ../ [%s]\n", "", parentDir)
	fmt.Printf("%"+pad+"s.) ./ [%s]\n", "", curDir)
	return nil
}

// drawSelBody draws selfs body
func drawSelBody(files []fs.FileInfo, min int, max int, pad string) (int, error) {
	lines := 0
	for num, file := range files {
		if num >= min && num <= max {
			symbol, err := utils.FileIndicator(file.Name())
			if err != nil {
				return -1, err
			}
			fmt.Printf(" %"+pad+"d) %s%s\n", num+1, file.Name(), symbol)
			lines++
		}
	}
	return lines, nil
}

// drawSelFooter draws selfs footer
func drawSelFooter(files []fs.FileInfo, min int, max int, show string, page int, pages int) error {
	var pos int
	if show == "first" {
		pos = min
	} else if show == "last" {
		pos = max
	}
	symbol, err := utils.FileIndicator(files[pos].Name())
	if err != nil {
		return err
	}
	fmt.Print("\n")
	fmt.Printf("# %d/%d) %s%s\n", pos+1, len(files), files[pos].Name(), symbol)
	fmt.Printf("> %d/%d", page, pages)
	return nil
}

// selfs selects a file using keyboard interactively
func selfs(files []fs.FileInfo) (string, error) {
	const linesHeader = 4
	const linesFooter = 3
	numPadInt := utils.CountDigit(len(files))
	numPadStr := strconv.Itoa(numPadInt)
	startOffset := 0
	if errSc := screen.Clear(); errSc != nil {
		return "", errSc
	}
	screenSize, errSs := screen.Size()
	if errSs != nil {
		return "", errSs
	}
	screenRows := screenSize[0]
	numPerPage := screenRows
	if numPerPage < linesHeader+linesFooter+1 {
		return "", errors.New("selfs: error: the terminal window is too small")
	}
	if errSc := screen.Clear(); errSc != nil {
		return "", errSc
	}
	if errDh := drawSelHeader(numPadStr); errDh != nil {
		return "", errDh
	}
	linesBody, errDb := drawSelBody(files, startOffset, numPerPage-(linesHeader+linesFooter+1), numPadStr)
	if errDb != nil {
		return "", errDb
	}
	if linesBody == 0 {
		return "", errors.New("selfs: error: no file was found to select")
	}
	numPage := 1
	numPages := int(math.Ceil(float64(len(files)) / float64(linesBody)))
	if errDf := drawSelFooter(files, 0, numPerPage-(linesHeader+linesFooter+1), "first", numPage, numPages); errDf != nil {
		return "", errDf
	}
	cursor.ResetModes()
	curPos := linesHeader + 1
	cursor.Move(curPos, numPadInt+1)
	for {
		key, errKp := utils.KeyPress()
		if errKp != nil {
			return "", errKp
		}
		keyName, errKn := utils.KeyPressName(key)
		if errKn != nil {
			return "", errKn
		}
		switch keyName {
		case "escape":
			return "", nil
		case "enter", "return":
			curFileName := files[(curPos+startOffset)-(linesHeader+1)].Name()
			return curFileName, nil
		case "J", "DOWN":
			curPos = linesHeader + linesBody
			cursor.Move(curPos, numPadInt+1)
		case "K", "UP":
			curPos = linesHeader + 1
			cursor.Move(curPos, numPadInt+1)
		case "j", "down":
			if curPos < linesHeader+linesBody {
				curPos++
				cursor.Move((linesHeader+linesBody+linesFooter)-1, 1)
				cursor.ClearCurLine()
				curFileName := files[(curPos+startOffset)-(linesHeader+1)].Name()
				symbol, errFi := utils.FileIndicator(curFileName)
				if errFi != nil {
					return "", errFi
				}
				fmt.Printf("# %d/%d) %s%s", (curPos+startOffset)-linesHeader, len(files), curFileName, symbol)
				cursor.Move(linesHeader+linesBody+linesFooter, 1)
				cursor.ClearCurLine()
				fmt.Printf("> %d/%d", numPage, numPages)
				cursor.Move(curPos, numPadInt+1)
			} else {
				if curPos >= len(files) && numPage >= numPages || numPage >= numPages {
					continue
				}
				if startOffset+(curPos-linesHeader) >= len(files) {
					return "", errors.New("selfs: error: startOffset number is bigger than the maximum number of files")
				}
				numPage++
				startOffset += curPos - linesHeader
				limitOffset := startOffset + numPerPage - (linesHeader + linesFooter + 1)
				if limitOffset > len(files) {
					limitOffset = len(files)
				}
				if errSc := screen.Clear(); errSc != nil {
					return "", errSc
				}
				if errDh := drawSelHeader(numPadStr); errDh != nil {
					return "", errDh
				}
				linesBody, errDb = drawSelBody(files, startOffset, limitOffset, numPadStr)
				if errDb != nil {
					return "", errDb
				}
				if errDf := drawSelFooter(files, startOffset, limitOffset, "first", numPage, numPages); errDf != nil {
					return "", errDf
				}
				curPos = linesHeader + 1
				cursor.Move(curPos, numPadInt+1)
			}
		case "k", "up":
			if curPos > linesHeader+1 {
				curPos--
				cursor.Move((linesHeader+linesBody+linesFooter)-1, 1)
				cursor.ClearCurLine()
				curFileName := files[(curPos+startOffset)-(linesHeader+1)].Name()
				symbol, errFi := utils.FileIndicator(curFileName)
				if errFi != nil {
					return "", errFi
				}
				fmt.Printf("# %d/%d) %s%s", (curPos+startOffset)-linesHeader, len(files), curFileName, symbol)
				cursor.Move(linesHeader+linesBody+linesFooter, 1)
				cursor.ClearCurLine()
				fmt.Printf("> %d/%d", numPage, numPages)
				cursor.Move(curPos, numPadInt+1)
			} else {
				if numPage <= 1 {
					continue
				}
				numPage--
				startOffset -= numPerPage - (linesHeader + linesFooter)
				limitOffset := (startOffset + numPerPage) - (linesHeader + linesFooter + 1)
				if errSc := screen.Clear(); errSc != nil {
					return "", errSc
				}
				if errDh := drawSelHeader(numPadStr); errDh != nil {
					return "", errDh
				}
				linesBody, errDb = drawSelBody(files, startOffset, limitOffset, numPadStr)
				if errDb != nil {
					return "", errDb
				}
				if errDf := drawSelFooter(files, startOffset, limitOffset, "last", numPage, numPages); errDf != nil {
					return "", errDf
				}
				curPos = numPerPage - linesFooter
				cursor.Move(curPos, numPadInt+1)
			}
		case "-":
			return "..", nil
		case ".":
			return ".", nil
		default:
			cursor.Move((linesHeader+linesBody+linesFooter)-1, 1)
			cursor.ClearCurLine()
			utils.ErrPrintf("# error: unsupported keystroke '%s'", keyName)
			cursor.Move(curPos, numPadInt)
		}
	}
}

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
	help += "selfs            # selects a file using keyboard interactively (j,k,J,K,Enter,Escape,-,.) [sel]\n"
	help += "stopplay         # stops playing the current media [stopp]\n"
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
			symbol, errFi := utils.FileIndicator(file.Name())
			if errFi != nil {
				return errFi
			}
			fmt.Printf("%s%"+numPad+"d) %s%s\n", selCur, num+1, file.Name(), symbol)
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
		case "stopp", "stopplay":
			curStream = ""
			fileSel = ""
			statusMsg = ""
			if err := PlayStop(); err != nil {
				statusMsg = err.Error()
			}
		case "sel", "selfs":
			var errMv error
			if fileStr, errMv = selfs(files); fileStr == "" || errMv != nil {
				if errMv != nil {
					statusMsg = errMv.Error()
				}
				continue
			}
			fallthrough
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
			if _, errSc := StatusCmd(cmd, "error", 1); errSc != nil {
				statusMsg = errSc.Error()
			}
		}
	}
}
