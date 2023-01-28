// by Gonzaru
// Distributed under the terms of the GNU General Public License v3

package sf

import (
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

// local packages
import (
	"github.com/gonzaru/gorum/config"
	"github.com/gonzaru/gorum/cursor"
	"github.com/gonzaru/gorum/gorum"
	"github.com/gonzaru/gorum/screen"
	"github.com/gonzaru/gorum/utils"
)

// selectFile data type
type selectFile struct {
	actionLoop  bool
	curPos      int
	files       []fs.FileInfo
	linesBody   int
	linesFooter int
	linesHeader int
	oldPwd      string
	padInt      int
	padStr      string
	page        int
	pages       int
	perPage     int
	progTitle   string
	pwd         string
	startOffset int
}

// helpSF shows sf' help information
func helpSF() string {
	var help strings.Builder
	help.WriteString("# help\n")
	help.WriteString(".       # lists the current directory contents\n")
	help.WriteString("-       # changes to parent directory\n")
	help.WriteString("_       # changes to previous directory [^,p]\n")
	help.WriteString("~       # changes to home user directory\n")
	help.WriteString("h       # goes to previous page\n")
	help.WriteString("l       # goes to next page\n")
	help.WriteString("j       # goes one line downward\n")
	help.WriteString("k       # goes one line upward\n")
	help.WriteString("J       # goes to bottom line\n")
	help.WriteString("K       # goes to top line\n")
	help.WriteString("r       # redraws terminal screen\n")
	help.WriteString("Enter   # selects the file or directory\n")
	help.WriteString("Escape  # exits sf [q]\n")
	help.WriteString("?       # shows sf' help information\n")
	return help.String()
}

// drawHeader draws sf header
func (sf *selectFile) drawHeader() error {
	pwdSplit := strings.Split(sf.pwd, "/")
	parentDir := pwdSplit[len(pwdSplit)-2]
	curDir := pwdSplit[len(pwdSplit)-1]
	fmt.Printf("%"+sf.padStr+"s### %s ###\n", "", strings.ToUpper(sf.progTitle))
	fmt.Printf("%"+sf.padStr+"s?) help\n", "")
	fmt.Printf("%"+sf.padStr+"s-) ../ [%s]\n", "", parentDir)
	fmt.Printf("%"+sf.padStr+"s.) ./ [%s]\n", "", curDir)
	return nil
}

// drawBody draws sf body
func (sf *selectFile) drawBody(min int, max int) (int, error) {
	lines := 0
	for num, file := range sf.files {
		if num >= min && num <= max {
			symbol, err := utils.FileIndicator(file.Name())
			if err != nil {
				return -1, err
			}
			fmt.Printf(" %"+sf.padStr+"d) %s%s\n", num+1, file.Name(), symbol)
			lines++
		}
	}
	return lines, nil
}

// drawFooter draws sf footer
func (sf *selectFile) drawFooter(pos int) error {
	if len(sf.files) > 0 {
		symbol, err := utils.FileIndicator(sf.files[pos].Name())
		if err != nil {
			return err
		}
		fmt.Print("\n")
		fmt.Printf("# %d/%d) %s%s\n", pos+1, len(sf.files), sf.files[pos].Name(), symbol)
		fmt.Printf("> %d/%d", sf.page, sf.pages)
	} else {
		fmt.Print("\n")
		fmt.Print("# empty directory, no files were found to select\n")
		fmt.Print("> ")
	}
	return nil
}

// nextLine goes one line downward
func (sf *selectFile) nextLine() error {
	sf.curPos++
	cursor.Move((sf.linesHeader+sf.linesBody+sf.linesFooter)-1, 1)
	cursor.ClearCurLine()
	curFileName := sf.files[(sf.curPos+sf.startOffset)-(sf.linesHeader+1)].Name()
	symbol, errFi := utils.FileIndicator(curFileName)
	if errFi != nil {
		return errFi
	}
	fmt.Printf("# %d/%d) %s%s", (sf.curPos+sf.startOffset)-sf.linesHeader, len(sf.files), curFileName, symbol)
	cursor.Move(sf.linesHeader+sf.linesBody+sf.linesFooter, 1)
	cursor.ClearCurLine()
	fmt.Printf("> %d/%d", sf.page, sf.pages)
	cursor.Move(sf.curPos, sf.padInt+1)
	return nil
}

// prevLine goes one line upward
func (sf *selectFile) prevLine() error {
	sf.curPos--
	cursor.Move((sf.linesHeader+sf.linesBody+sf.linesFooter)-1, 1)
	cursor.ClearCurLine()
	curFileName := sf.files[(sf.curPos+sf.startOffset)-(sf.linesHeader+1)].Name()
	symbol, errFi := utils.FileIndicator(curFileName)
	if errFi != nil {
		return errFi
	}
	fmt.Printf("# %d/%d) %s%s", (sf.curPos+sf.startOffset)-sf.linesHeader, len(sf.files), curFileName, symbol)
	cursor.Move(sf.linesHeader+sf.linesBody+sf.linesFooter, 1)
	cursor.ClearCurLine()
	fmt.Printf("> %d/%d", sf.page, sf.pages)
	cursor.Move(sf.curPos, sf.padInt+1)
	return nil
}

// nextPage goes to next page
func (sf *selectFile) nextPage() error {
	if sf.curPos >= len(sf.files) && sf.page >= sf.pages || sf.page >= sf.pages {
		return nil
	}
	if sf.startOffset+(sf.curPos-sf.linesHeader) >= len(sf.files) {
		return errors.New("nextPage: error: startOffset number is bigger than the maximum number of files")
	}
	sf.page++
	sf.startOffset += sf.curPos - sf.linesHeader
	limitOffset := sf.startOffset + sf.perPage - (sf.linesHeader + sf.linesFooter + 1)
	if limitOffset > len(sf.files) {
		limitOffset = len(sf.files)
	}
	if errSc := screen.Clear(); errSc != nil {
		return errSc
	}
	if errDh := sf.drawHeader(); errDh != nil {
		return errDh
	}
	var errDb error
	sf.linesBody, errDb = sf.drawBody(sf.startOffset, limitOffset)
	if errDb != nil {
		return errDb
	}
	if errDf := sf.drawFooter(sf.startOffset); errDf != nil {
		return errDf
	}
	sf.curPos = sf.linesHeader + 1
	cursor.Move(sf.curPos, sf.padInt+1)
	return nil
}

// prevPage goes to previous page
func (sf *selectFile) prevPage(curTop bool) error {
	if sf.page <= 1 {
		return nil
	}
	sf.page--
	sf.startOffset -= sf.perPage - (sf.linesHeader + sf.linesFooter)
	limitOffset := (sf.startOffset + sf.perPage) - (sf.linesHeader + sf.linesFooter + 1)
	if errSc := screen.Clear(); errSc != nil {
		return errSc
	}
	if errDh := sf.drawHeader(); errDh != nil {
		return errDh
	}
	var errDb error
	sf.linesBody, errDb = sf.drawBody(sf.startOffset, limitOffset)
	if errDb != nil {
		return errDb
	}
	if errDf := sf.drawFooter(sf.startOffset); errDf != nil {
		return errDf
	}
	sf.curPos = sf.linesHeader + sf.linesBody
	if curTop {
		sf.curPos = sf.linesHeader + 1
	}
	cursor.Move(sf.curPos, sf.padInt+1)
	return nil
}

// doActionEnter executes the enter sf option
func (sf *selectFile) doActionEnter() error {
	if len(sf.files) == 0 || len(sf.files) <= (sf.curPos+sf.startOffset)-(sf.linesHeader+1) {
		return nil
	}
	curFileName := sf.files[(sf.curPos+sf.startOffset)-(sf.linesHeader+1)]
	curFileIsDir := false
	if curFileName.Mode()&os.ModeSymlink == os.ModeSymlink {
		symlinkPath, errRl := os.Readlink(curFileName.Name())
		if errRl != nil {
			return errRl
		}
		fi, errOs := os.Stat(symlinkPath)
		if os.IsNotExist(errOs) {
			return fmt.Errorf("doActionEnter: error: '%s' no such file or directory\n", symlinkPath)
		} else if errOs != nil {
			return errOs
		}
		if fi.IsDir() {
			curFileIsDir = true
		}
	}
	if curFileName.IsDir() || curFileIsDir {
		if errCd := os.Chdir(curFileName.Name()); errCd != nil {
			return errCd
		}
		sf.oldPwd = sf.pwd
		sf.actionLoop = false
	} else {
		if errPl := gorum.Play(curFileName.Name()); errPl != nil {
			return errPl
		}
		cmd := `{"command": ["get_property", "filtered-metadata"]}`
		if _, errSc := gorum.StatusCmd(cmd, "error", config.MinStatusTries); errSc != nil {
			log.Print(errSc)
			cursor.Move((sf.linesHeader+sf.linesBody+sf.linesFooter)-1, 1)
			cursor.ClearCurLine()
			utils.ErrPrintf("# %s", errSc.Error())
			cursor.Move(sf.curPos, sf.padInt+1)
		}
	}
	return nil
}

// doActionHelp executes the help sf option
func (sf *selectFile) doActionHelp() error {
	cursor.Move((sf.linesHeader+sf.linesBody+sf.linesFooter)-1, 1)
	cursor.ClearCurLine()
	fmt.Print(helpSF())
	fmt.Print("\nPress ENTER to exit")
	res := ""
	if _, errSc := fmt.Scanf("%s", &res); errSc != nil && errSc.Error() != "unexpected newline" {
		return errSc
	}
	cursor.Move(sf.linesHeader+1, sf.padInt+1)
	sf.actionLoop = false
	return nil
}

// doActionHomeDir executes the home dir sf option
func (sf *selectFile) doActionHomeDir() error {
	homeDir, errUh := os.UserHomeDir()
	if errUh != nil {
		return errUh
	}
	if errCd := os.Chdir(homeDir); errCd != nil {
		return errCd
	}
	sf.oldPwd = sf.pwd
	sf.actionLoop = false
	return nil
}

// doActionDownLine executes the down line sf option
func (sf *selectFile) doActionDownLine() error {
	if sf.curPos < sf.linesHeader+sf.linesBody {
		if errNl := sf.nextLine(); errNl != nil {
			return errNl
		}
	} else {
		if errNp := sf.nextPage(); errNp != nil {
			return errNp
		}
	}
	return nil
}

// doActionParentDir executes the parent dir sf option
func (sf *selectFile) doActionParentDir() error {
	if sf.pwd != "/" {
		if errCd := os.Chdir(".."); errCd != nil {
			return errCd
		}
		sf.oldPwd = sf.pwd
		sf.actionLoop = false
	}
	return nil
}

// doActionPrevDir executes the previous dir sf option
func (sf *selectFile) doActionPrevDir() error {
	if sf.oldPwd != "" && sf.oldPwd != sf.pwd {
		if errCd := os.Chdir(sf.oldPwd); errCd != nil {
			return errCd
		}
		sf.oldPwd = sf.pwd
		sf.actionLoop = false
	}
	return nil
}

// doActionUpLine executes the up line sf option
func (sf *selectFile) doActionUpLine() error {
	if sf.curPos > sf.linesHeader+1 {
		if errPl := sf.prevLine(); errPl != nil {
			return errPl
		}
	} else {
		if errPp := sf.prevPage(false); errPp != nil {
			return errPp
		}
	}
	return nil
}

// doAction executes the selected sf option
func (sf *selectFile) doAction(keyName string) error {
	switch keyName {
	case "?":
		if err := sf.doActionHelp(); err != nil {
			return err
		}
	case "_", "^", "p":
		if err := sf.doActionPrevDir(); err != nil {
			return err
		}
	case "-":
		if err := sf.doActionParentDir(); err != nil {
			return err
		}
	case "~":
		if err := sf.doActionHomeDir(); err != nil {
			return err
		}
	case ".":
		sf.actionLoop = false
	case "escape", "q":
		sf.actionLoop = false
	case "enter", "return":
		if err := sf.doActionEnter(); err != nil {
			return err
		}
	case "J", "DOWN":
		sf.curPos = sf.linesHeader + sf.linesBody
		cursor.Move(sf.curPos, sf.padInt+1)
	case "K", "UP":
		sf.curPos = sf.linesHeader + 1
		cursor.Move(sf.curPos, sf.padInt+1)
	case "j", "down":
		if err := sf.doActionDownLine(); err != nil {
			return err
		}
	case "k", "up":
		if err := sf.doActionUpLine(); err != nil {
			return err
		}
	case "h", "left":
		if sf.pages > 1 {
			if errNp := sf.prevPage(true); errNp != nil {
				return errNp
			}
		}
	case "l", "right":
		if sf.pages > 1 {
			sf.curPos = sf.linesHeader + sf.linesBody
			if errNp := sf.nextPage(); errNp != nil {
				return errNp
			}
		}
	case "r":
		sf.actionLoop = false
	default:
		cursor.Move((sf.linesHeader+sf.linesBody+sf.linesFooter)-1, 1)
		cursor.ClearCurLine()
		utils.ErrPrintf("# sf: error: keystroke '%s' is not supported, press '?' for help", keyName)
		cursor.Move(sf.curPos, sf.padInt+1)
	}
	return nil
}

// runActions runs the action loop
func (sf *selectFile) runActions() error {
	for sf.actionLoop = true; sf.actionLoop; {
		key, errKp := utils.KeyPress()
		if errKp != nil {
			return errKp
		}
		keyName, errKn := utils.KeyPressName(key)
		if errKn != nil {
			return errKn
		}
		if keyName == "escape" || keyName == "q" {
			return errors.New("info: 'sf' was closed")
		}
		if errDa := sf.doAction(keyName); errDa != nil {
			return errDa
		}
	}
	return nil
}

// Run selects a file using keyboard interactively
func Run() error {
	if !gorum.IsRunning() {
		return fmt.Errorf("info: error: '%s' is not running\n", config.ProgName)
	}
	sf := selectFile{
		linesHeader: 4,
		linesFooter: 3,
		progTitle:   config.ProgName,
	}
	var errAl, errDb, errOg, errRd error
	for {
		sf.pwd, errOg = os.Getwd()
		if errOg != nil {
			return errOg
		}
		sf.files, errRd = ioutil.ReadDir(sf.pwd)
		if errRd != nil {
			return errRd
		}
		sf.padInt = utils.CountDigit(len(sf.files))
		sf.padStr = strconv.Itoa(sf.padInt)
		if errSc := screen.Clear(); errSc != nil {
			return errSc
		}
		screenSize, errSs := screen.Size()
		if errSs != nil {
			return errSs
		}
		sf.perPage = screenSize[0]
		if sf.perPage < sf.linesHeader+sf.linesFooter+1 {
			return errors.New("sf: error: the terminal window is too small")
		}
		if errDh := sf.drawHeader(); errDh != nil {
			return errDh
		}
		sf.startOffset = 0
		sf.linesBody, errDb = sf.drawBody(sf.startOffset, sf.perPage-(sf.linesHeader+sf.linesFooter+1))
		if errDb != nil {
			return errDb
		}
		sf.page = 1
		sf.pages = int(math.Ceil(float64(len(sf.files)) / float64(sf.linesBody)))
		if errDf := sf.drawFooter(sf.startOffset); errDf != nil {
			return errDf
		}
		sf.curPos = sf.linesHeader + 1
		cursor.ResetModes()
		cursor.Move(sf.curPos, sf.padInt+1)
		errAl = sf.runActions()
		if errAl != nil {
			return errAl
		}
	}
}
