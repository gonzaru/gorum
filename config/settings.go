// by Gonzaru
// Distributed under the terms of the GNU General Public License v3

package config

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
)

// ProgName the name of the program
const ProgName = "gorum"

var (
	LockDir    = fmt.Sprintf("%s/%s-%s.lock", tmpDir, userName, ProgName)
	PidFile    = fmt.Sprintf("%s/%s-%s.pid", tmpDir, userName, ProgName)
	Player     = "mpv"
	PlayerArgs = []string{
		"--no-config",
		"--msg-level=all=v",
		"--network-timeout=10",
		"--cache=no",
		"--cache-pause=no",
		"--keep-open=always",
		"--keep-open-pause=no",
		"--idle=yes",
		"--input-ipc-server=" + PlayerControlFile,
	}
	PlayerControlFile = fmt.Sprintf("%s/%s-%s-player-control.socket", tmpDir, userName, ProgName)
	PlayerPidFile     = fmt.Sprintf("%s/%s-%s-player.pid", tmpDir, userName, ProgName)
	Log               = fmt.Sprintf("%s/%s-%s.log", tmpDir, userName, ProgName)
	MaxMenuTries      = 5
	MinStatusTries    = 1
	MaxStatusTries    = 10
	VolumeMin         = 0
	VolumeMax         = 100
	VolumeAbsolute    = 100
	WmDoBarUpdate     = wmCheckBarUpdate("wmbarupdate")
	WmFile            = fmt.Sprintf("%s/%s-%s-wm.txt", tmpDir, userName, ProgName)
	WmFilePerms       = os.FileMode(0600)
	tmpDir            = os.TempDir()
	userName          = getUserName()
)

// getUserName returns the current user name
func getUserName() string {
	usc, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usc.Username
}

// wmCheckBarUpdate checks if wmbarupdate command exists
func wmCheckBarUpdate(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
