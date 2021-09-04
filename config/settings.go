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
	LockDir           = fmt.Sprintf("%s/%s-gorum.lock", tmpDir, userName)
	PidFile           = fmt.Sprintf("%s/%s-gorum.pid", tmpDir, userName)
	Player            = "mpv"
	PlayerArgs        = []string{"--msg-level=all=v", "--idle=yes", "--input-ipc-server=" + PlayerControlFile}
	PlayerControlFile = fmt.Sprintf("%s/%s-gorum-player-control.socket", tmpDir, userName)
	PlayerPidFile     = fmt.Sprintf("%s/%s-gorum-player.pid", tmpDir, userName)
	GorumLog          = fmt.Sprintf("%s/%s-gorum.log", tmpDir, userName)
	WmDoBarUpdate     = wmCheckBarUpdate("wmbarupdate")
	WmFile            = fmt.Sprintf("%s/%s-gorum-wm.txt", tmpDir, userName)
	WmFilePerms       = os.FileMode(0600)
	tmpDir            = getTmpDir()
	userName          = getUserName()
)

// getTmpDir
func getTmpDir() string {
	var defTmpDir = "/tmp"
	userTmpDir := os.Getenv("TMPDIR")
	if userTmpDir != "" {
		defTmpDir = userTmpDir
	}
	return defTmpDir
}

// getUserName
func getUserName() string {
	usc, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usc.Username
}

// wmCheckBarUpdate
func wmCheckBarUpdate(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
