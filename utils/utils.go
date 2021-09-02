// by Gonzaru
// Distributed under the terms of the GNU General Public License v3

package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// CountDigit counts the number of digits in a number
func CountDigit(num int) int {
	count := 0
	for num != 0 {
		num /= 10
		count++
	}
	return count
}

// ErrPrint prints error message to stderr using the default formats
func ErrPrint(v ...interface{}) {
	if _, err := fmt.Fprint(os.Stderr, v...); err != nil {
		log.Fatal(err)
	}
}

// ErrPrintf prints error message to stderr according to a format specifier
func ErrPrintf(format string, v ...interface{}) {
	if _, err := fmt.Fprintf(os.Stderr, format, v...); err != nil {
		log.Fatal(err)
	}
}

// FileIndicator returns an indicator that identifies a file (*/=@|)
func FileIndicator(file string) (string, error) {
	var symbol string
	fi, err := os.Lstat(file)
	if err != nil {
		return "", err
	}
	if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		symbol = "@"
	} else if fi.IsDir() {
		symbol = "/"
	} else if fi.Mode()&0111 != 0 {
		symbol = "*"
	} else if fi.Mode()&os.ModeNamedPipe != 0 {
		symbol = "|"
	} else if fi.Mode()&os.ModeSocket != 0 {
		symbol = "="
	} else {
		symbol = ""
	}
	return symbol, nil
}

// JsonPretty returns json with a more readable format
func JsonPretty(dataJson []byte, prefix string, delim string) (string, error) {
	var dataPretty bytes.Buffer
	if !json.Valid(dataJson) {
		return "", errors.New(fmt.Sprintf("JsonPretty: error: invalid json %s\n", dataJson))
	}
	if err := json.Indent(&dataPretty, dataJson, prefix, delim); err != nil {
		return "", err
	}
	return dataPretty.String(), nil
}

// KeyPress gets the pressed key
func KeyPress() ([]byte, error) {
	key := make([]byte, 3, 3)
	fileFlag := "-f"
	if runtime.GOOS == "linux" {
		fileFlag = "-F"
	}
	if errCs := exec.Command("stty", fileFlag, "/dev/tty", "cbreak", "min", "1").Run(); errCs != nil {
		return nil, errCs
	}
	if errCs := exec.Command("stty", fileFlag, "/dev/tty", "-echo").Run(); errCs != nil {
		return nil, errCs
	}
	defer func() {
		if errCs := exec.Command("stty", fileFlag, "/dev/tty", "echo").Run(); errCs != nil {
			ErrPrint(errCs)
			log.Fatal(errCs)
		}
		cmdSs := exec.Command("stty", "sane")
		cmdSs.Stdin = os.Stdin
		if errCr := cmdSs.Run(); errCr != nil {
			ErrPrint(errCr)
			log.Fatal(errCr)
		}
	}()
	if _, errSr := os.Stdin.Read(key); errSr != nil {
		return nil, errSr
	}
	return key, nil
}

// KeyPressName returns the name of pressed key
func KeyPressName(key []byte) (string, error) {
	var keyName string
	keySize := 3
	if len(key) != keySize {
		return "", errors.New(fmt.Sprintf("keyPressName: error: key needs to be size %d", keySize))
	}
	if key[0] != 0 && key[1] == 0 && key[2] == 0 {
		if key[0] == 27 {
			keyName = "escape"
		} else if key[0] == 10 {
			keyName = "enter"
		} else {
			keyName = string(key[0])
		}
	} else if key[0] == 27 && key[1] == 91 && key[2] == 65 {
		keyName = "up"
	} else if key[0] == 27 && key[1] == 91 && key[2] == 66 {
		keyName = "down"
	} else if key[0] == 59 && key[1] == 50 && key[2] == 65 { // <S-Up>
		keyName = "UP"
	} else if key[0] == 59 && key[1] == 50 && key[2] == 66 { // <S-Down>
		keyName = "DOWN"
	} else {
		keyName = string(key)
	}
	return keyName, nil
}

// PidFileExists checks if file and pid exists
func PidFileExists(file string) bool {
	status := false
	if _, errSt := os.Stat(file); errSt != nil {
		return false
	}
	content, errRf := ioutil.ReadFile(file)
	if errRf != nil {
		return false
	}
	pid := strings.TrimRight(string(content), "\n")
	pidPath := "/proc/" + pid
	if fi, errOs := os.Stat(pidPath); errOs == nil && fi.IsDir() {
		status = true
	}
	return status
}

// ValidUrl checks if is a valid url format
func ValidUrl(str string) bool {
	status := false
	if u, err := url.Parse(str); err == nil && u.Scheme != "" && u.Host != "" && u.Path != "" {
		status = true
	}
	return status
}
