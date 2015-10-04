package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/howeyc/fsnotify"
)

// parseArguments ...
func parseArguments(ev *fsnotify.FileEvent) (cmd string, args []string) {
	commandline := strings.Fields(flag.Args()[0])
	cmd = commandline[0]
	args = make([]string, len(commandline[1:]))

	// Do file name interpolation on the arguments if %f is in them
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

	return
}

func handle(ev *fsnotify.FileEvent) {
	command, args := parseArguments(ev)
	cmd := exec.Command(command, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	// Clear the screen
	fmt.Print("\033[H\033[2J")

	yellow := color.New(color.FgYellow, color.Bold).SprintfFunc()
	magenta := color.New(color.FgMagenta, color.Bold).SprintfFunc()
	red := color.New(color.FgRed, color.Bold).SprintfFunc()
	green := color.New(color.FgGreen, color.Bold).SprintfFunc()

	// Print the command in nice colors
	out := fmt.Sprintf("Running %s %s...", yellow(command), magenta(strings.Join(args, " ")))

	log.Println(out)
	if err = cmd.Start(); err != nil {
		log.Fatal(err)
	}

	outchan := make(chan string)
	errchan := make(chan string)
	outdonechan := make(chan bool)
	errdonechan := make(chan bool)
	outdone := false

	go loopOutput(outchan, outdonechan, stdout)
	go loopOutput(errchan, errdonechan, stderr)

	for outdone == false {
		select {
		case msg := <-outchan:
			fmt.Print(msg)
		case errmsg := <-errchan:
			fmt.Print(red(errmsg))
		case <-outdonechan:
			outdone = true
		}
	}

	// Print red error message or green success message
	if err = cmd.Wait(); err != nil {
		log.Println(red(err.Error()))
	} else {
		log.Println(green("Execution successful."))
	}
}

func loopOutput(c chan string, done chan bool, pipe io.ReadCloser) {
	buf := make([]byte, 1024)

	for {
		n, err := pipe.Read(buf)

		// Either if the pipe was empty or an EOF or other error was returned.
		if n == 0 && err == nil || err != nil {
			done <- true
			return
		}

		c <- string(buf[:n])
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
