// by Gonzaru
// Distributed under the terms of the GNU General Public License v3

package screen

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Clear clears the entire terminal screen
func Clear() error {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// Size obtains the terminal number of rows and columns
func Size() ([]int, error) {
	size := make([]int, 2)
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	content, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	contentSplit := strings.Fields(string(content))
	if len(contentSplit) == 0 {
		return nil, errors.New("size: error: terminal's number of rows and columns was not found")
	}
	rows, errSa := strconv.Atoi(contentSplit[0])
	if errSa != nil {
		return nil, errSa
	}
	cols, errSa := strconv.Atoi(contentSplit[1])
	if errSa != nil {
		return nil, errSa
	}
	size[0] = rows
	size[1] = cols
	return size, nil
}
