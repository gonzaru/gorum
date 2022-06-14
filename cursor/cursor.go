// by Gonzaru
// Distributed under the terms of the GNU General Public License v3

package cursor

import "fmt"

// Escape ASCII hexadecimal escape character
const Escape = "\x1b"

// ClearCurLine clears the current line
func ClearCurLine() {
	fmt.Printf("%s[K", Escape)
}

// Move moves the cursor at line {line}, column {col}
func Move(line int, col int) {
	fmt.Printf("%s[%d;%dH", Escape, line, col)
}

// ResetModes resets all modes
func ResetModes() {
	fmt.Printf("%s[0m", Escape)
}
