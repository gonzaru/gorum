// by Gonzaru
// Distributed under the terms of the GNU General Public License v3

package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
)

// local packages
import (
	"github.com/gonzaru/gorum/config"
	"github.com/gonzaru/gorum/gorum"
	"github.com/gonzaru/gorum/menu"
	"github.com/gonzaru/gorum/utils"
)

// main options
func main() {
	if errSl := gorum.SetLog(); errSl != nil {
		utils.ErrPrint(errSl)
		log.Fatal(errSl)
	}
	if errCo := gorum.CheckOut(); errCo != nil {
		utils.ErrPrint(errCo)
		log.Fatal(errCo)
	}
	args := os.Args[1:]
	lenArgs := len(args)
	if lenArgs == 0 {
		gorum.Help()
		os.Exit(1)
	}
	arg := args[0]
	switch arg {
	case "check":
		if !gorum.IsRunning() {
			utils.ErrPrintf("main: error: '%s' is not running\n", config.ProgName)
			os.Exit(1)
		}
	case "help":
		gorum.Help()
	case "menu":
		go menu.SignalHandler()
		if errMe := menu.Menu(); errMe != nil {
			utils.ErrPrint(errMe)
			log.Fatal(errMe)
		}
	case "mute", "pause", "video":
		if err := gorum.Toggle(arg); err != nil {
			utils.ErrPrint(err)
			log.Fatal(err)
		}
	case "seek":
		if len(args) != 2 {
			gorum.Help()
			os.Exit(1)
		}
		secondsStr := args[1]
		regexSeek := regexp.MustCompile(`^[+-]\d+$`)
		if match := regexSeek.MatchString(secondsStr); !match {
			gorum.Help()
			os.Exit(1)
		}
		secondsInt, errSa := strconv.Atoi(secondsStr)
		if errSa != nil {
			utils.ErrPrint(errSa)
			log.Fatal(errSa)
		}
		if errSe := gorum.Seek(secondsInt); errSe != nil {
			utils.ErrPrint(errSe)
			log.Fatal(errSe)
		}
	case "start":
		go gorum.SignalHandler()
		if err := gorum.Start(); err != nil {
			utils.ErrPrint(err)
			log.Fatal(err)
		}
	case "status":
		content, err := gorum.Status()
		if err != nil {
			utils.ErrPrint(err)
			log.Fatal(err)
		}
		fmt.Print(content)
	case "stop":
		if err := gorum.Stop(); err != nil {
			utils.ErrPrint(err)
			log.Fatal(err)
		}
	case "stopp", "stopplay":
		if err := gorum.PlayStop(); err != nil {
			utils.ErrPrint(err)
			log.Fatal(err)
		}
	case "title":
		content, err := gorum.Title()
		if err != nil {
			utils.ErrPrint(err)
			log.Fatal(err)
		}
		fmt.Printf("title: %s\n", content)
	case "vol", "volume":
		if len(args) != 2 {
			gorum.Help()
			os.Exit(1)
		}
		numStr := args[1]
		numInt, errSa := strconv.Atoi(numStr)
		if errSa != nil {
			utils.ErrPrint(errSa)
			log.Fatal(errSa)
		}
		if errVo := gorum.Volume(numInt); errVo != nil {
			utils.ErrPrint(errVo)
			log.Fatal(errVo)
		}
	default:
		if err := gorum.Play(arg); err != nil {
			utils.ErrPrint(err)
			log.Fatal(err)
		}
		cmd := `{"command": ["get_property", "filtered-metadata"]}`
		if _, errSc := gorum.StatusCmd(cmd, "error", config.MaxStatusTries); errSc != nil {
			utils.ErrPrint(errSc)
			log.Fatal(errSc)
		}
	}
}
