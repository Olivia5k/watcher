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
	args := make([]string, len(commandline[1:]))

	// Figure out if we should do file name interpolation on the arguments
	for index, arg := range commandline[1:] {
		var err error
		if strings.Contains(arg, "%f") {
			arg, err = filepath.Abs(strings.Replace(arg, "%f", ev.Name, -1))
			if err != nil {
				log.Fatal(err)
			}
		}
		args[index] = arg
	}

	cmd := exec.Command(commandline[0], args...)

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("\033[H\033[2J") // Clear the screen
	// Print the command in nice colors
	yellow := color.New(color.FgYellow, color.Bold).SprintfFunc()
	magenta := color.New(color.FgMagenta, color.Bold).SprintfFunc()
	out := fmt.Sprintf("Running %s %s...", yellow(commandline[0]), magenta(strings.Join(args, " ")))

	log.Println(out)
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

		fmt.Print(string(buff[:n]))
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

	log.Printf("Watching %s...", *dir)
	err = watcher.Watch(*dir)
	if err != nil {
		log.Fatal(err)
	}

	<-done
}
