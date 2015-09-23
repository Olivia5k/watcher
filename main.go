package main

import (
	"flag"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/howeyc/fsnotify"
)

func handle(ev *fsnotify.FileEvent) {
	commandline := strings.Fields(flag.Args()[0])
	cmd := exec.Command(commandline[0], commandline[1:]...)

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("\033[H\033[2J") // Clear the screen
	log.Println(fmt.Sprintf("Running %s...", commandline[0]))
	if err = cmd.Start(); err != nil {
		log.Fatal(err)
	}

	buff := make([]byte, 1024)

	for {
		n, err := pipe.Read(buff)
		// Either if the pipe was empty or an EOF or other error was returned.
		if n == 0 && err == nil || err != nil {
			break
		}

		s := string(buff[:n])
		fmt.Print(s)
	}

	// Print red error message or green success message
	if err = cmd.Wait(); err != nil {
		red := color.New(color.FgRed, color.Bold).SprintfFunc()
		log.Println(red(err.Error()))
	} else {
		green := color.New(color.FgGreen, color.Bold).SprintfFunc()
		log.Println(green("Execution successful."))
	}
}

func main() {
	dir := flag.String("d", ".", "directory to watch")
	flag.Parse()

	if flag.NArg() != 1 {
		log.Fatal("One command line argument sting required.")
	}

	absdir, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatal(err)
	}

	watcher, err := fsnotify.NewWatcher()
	defer watcher.Close()

	if err != nil {
		log.Fatal(err)
	}

	done := make(chan bool)

	// Process events
	go func() {
		for {
			select {
			case ev := <-watcher.Event:
				handle(ev)

			case err := <-watcher.Error:
				log.Println("error:", err)

			}
		}
	}()

	log.Printf("Watching %s...", absdir)
	err = watcher.Watch(*dir)
	if err != nil {
		log.Fatal(err)
	}

	<-done
}
