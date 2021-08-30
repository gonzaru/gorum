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

// PidFileExists checks if file and pid exists
func PidFileExists(file string) bool {
	var status = false
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
	var status = false
	if u, err := url.Parse(str); err == nil && u.Scheme != "" && u.Host != "" && u.Path != "" {
		status = true
	}
	return status
}
