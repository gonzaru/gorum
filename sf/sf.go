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
	files       []fs.FileInfo
	padStr      string
	pwd         string
	oldPwd      string
	curPos      int
	linesHeader int
	linesBody   int
	linesFooter int
	padInt      int
	page        int
	pages       int
	perPage     int
	startOffset int
}

// helpSF shows sf' help information
func helpSF() string {
	help := "# help\n"
	help += ".       # lists the current directory contents\n"
	help += "-       # changes to parent directory\n"
	help += "_       # changes to previous directory [^,p]\n"
	help += "h       # goes to previous page\n"
	help += "l       # goes to next page\n"
	help += "j       # goes one line downward\n"
	help += "k       # goes one line upward\n"
	help += "J       # goes to bottom line\n"
	help += "K       # goes to top line\n"
	help += "r       # redraws terminal screen\n"
	help += "Enter   # selects the file or directory\n"
	help += "Escape  # exits sf\n"
	help += "=       # shows sf' help information\n"
	return help
}

// drawHeader draws sf header
func (sf *selectFile) drawHeader() error {
	pwdSplit := strings.Split(sf.pwd, "/")
	parentDir := pwdSplit[len(pwdSplit)-2]
	curDir := pwdSplit[len(pwdSplit)-1]
	fmt.Printf("%"+sf.padStr+"s### %s ###\n", "", strings.ToUpper(config.ProgName))
	fmt.Printf("%"+sf.padStr+"s=) help (j,k,Enter,Escape)\n", "")
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

// nextLine
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

// prevLine
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

// nextPage
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

// prevPage
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

// Run selects a file using keyboard interactively
func Run() error {
	sf := selectFile{
		linesHeader: 4,
		linesFooter: 3,
	}
	for {
		var errOg error
		sf.pwd, errOg = os.Getwd()
		if errOg != nil {
			return errOg
		}
		var errRd error
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
		var errDb error
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
		for keyLoop := true; keyLoop; {
			key, errKp := utils.KeyPress()
			if errKp != nil {
				return errKp
			}
			keyName, errKn := utils.KeyPressName(key)
			if errKn != nil {
				return errKn
			}
			switch keyName {
			case "=":
				cursor.Move((sf.linesHeader+sf.linesBody+sf.linesFooter)-1, 1)
				cursor.ClearCurLine()
				fmt.Print(helpSF())
				fmt.Print("\nPress ENTER to exit")
				res := ""
				if _, errSc := fmt.Scanf("%s", &res); errSc != nil && errSc.Error() != "unexpected newline" {
					return errSc
				}
				cursor.Move(sf.linesHeader+1, sf.padInt+1)
				keyLoop = false
			case "_", "^", "p":
				if sf.oldPwd != "" && sf.oldPwd != sf.pwd {
					if errCd := os.Chdir(sf.oldPwd); errCd != nil {
						return errCd
					}
					sf.oldPwd = sf.pwd
					keyLoop = false
				}
			case "-":
				if sf.pwd != "/" {
					if errCd := os.Chdir(".."); errCd != nil {
						return errCd
					}
					sf.oldPwd = sf.pwd
					keyLoop = false
				}
			case ".":
				keyLoop = false
			case "escape":
				return nil
			case "enter", "return":
				if len(sf.files) == 0 || len(sf.files) <= (sf.curPos+sf.startOffset)-(sf.linesHeader+1) {
					continue
				}
				curFileName := sf.files[(sf.curPos+sf.startOffset)-(sf.linesHeader+1)]
				curFileIsDir := false
				if curFileName.Mode()&os.ModeSymlink == os.ModeSymlink {
					symlinkPath, errRl := os.Readlink(curFileName.Name())
					if errRl != nil {
						return errRl
					}
					fi, errOs := os.Stat(symlinkPath)
					if errOs != nil {
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
					keyLoop = false
				} else {
					if errPl := gorum.Play(curFileName.Name()); errPl != nil {
						return errPl
					}
					cmd := `{"command": ["get_property", "filtered-metadata"]}`
					if _, errSc := gorum.StatusCmd(cmd, "error", 1); errSc != nil {
						log.Print(errSc)
						cursor.Move((sf.linesHeader+sf.linesBody+sf.linesFooter)-1, 1)
						cursor.ClearCurLine()
						utils.ErrPrintf("# %s", errSc.Error())
						cursor.Move(sf.curPos, sf.padInt+1)
					}
				}
			case "J", "DOWN":
				sf.curPos = sf.linesHeader + sf.linesBody
				cursor.Move(sf.curPos, sf.padInt+1)
			case "K", "UP":
				sf.curPos = sf.linesHeader + 1
				cursor.Move(sf.curPos, sf.padInt+1)
			case "j", "down":
				if sf.curPos < sf.linesHeader+sf.linesBody {
					if errNl := sf.nextLine(); errNl != nil {
						return errNl
					}
				} else {
					if errNp := sf.nextPage(); errNp != nil {
						return errNp
					}
				}
			case "k", "up":
				if sf.curPos > sf.linesHeader+1 {
					if errPl := sf.prevLine(); errPl != nil {
						return errPl
					}
				} else {
					if errPp := sf.prevPage(false); errPp != nil {
						return errPp
					}
				}
			case "h":
				if sf.pages > 1 {
					if errNp := sf.prevPage(true); errNp != nil {
						return errNp
					}
				}
			case "l":
				if sf.pages > 1 {
					sf.curPos = sf.linesHeader + sf.linesBody
					if errNp := sf.nextPage(); errNp != nil {
						return errNp
					}
				}
			case "r":
				keyLoop = false
			default:
				cursor.Move((sf.linesHeader+sf.linesBody+sf.linesFooter)-1, 1)
				cursor.ClearCurLine()
				utils.ErrPrintf("# error: unsupported keystroke '%s'", keyName)
				cursor.Move(sf.curPos, sf.padInt+1)
			}
		}
	}
}
