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

const ProgName = "gorum"

var (
	LockDir    = fmt.Sprintf("%s/%s-gorum.lock", tmpDir, userName)
	PidFile    = fmt.Sprintf("%s/%s-gorum.pid", tmpDir, userName)
	Player     = "mpv"
	PlayerArgs = []string{
		"--msg-level=all=v",
		"--network-timeout=10",
		"--cache=no",
		"--cache-pause=no",
		"--keep-open=always",
		"--keep-open-pause=no",
		"--idle=yes",
		"--input-ipc-server=" + PlayerControlFile,
	}
	PlayerControlFile = fmt.Sprintf("%s/%s-gorum-player-control.socket", tmpDir, userName)
	PlayerPidFile     = fmt.Sprintf("%s/%s-gorum-player.pid", tmpDir, userName)
	GorumLog          = fmt.Sprintf("%s/%s-gorum.log", tmpDir, userName)
	VolumeMin         = 0
	VolumeMax         = 100
	VolumeAbsolute    = 100
	WmDoBarUpdate     = wmCheckBarUpdate("wmbarupdate")
	WmFile            = fmt.Sprintf("%s/%s-gorum-wm.txt", tmpDir, userName)
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
