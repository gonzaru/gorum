// by Gonzaru
// Distributed under the terms of the GNU General Public License v3

package main

import (
	"fmt"
	"log"
	"os"
)

// local packages
import (
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
	default:
		if err := gorum.Play(arg); err != nil {
			utils.ErrPrint(err)
			log.Fatal(err)
		}
		cmd := `{"command": ["get_property", "filtered-metadata"]}`
		if _, errSc := gorum.StatusCmd(cmd, "error", 5); errSc != nil {
			utils.ErrPrint(errSc)
			log.Fatal(errSc)
		}
	}
}
