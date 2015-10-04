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

// parseArguments parses the command line flags and does interpolation
// of special formatting strings, like %f for the file name of the
// file that triggered the FileEvent.
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
				log.Fatal("Could not get absolute path: ", err)
			}
		}
		args[index] = arg
	}

	return
}

// handle executes an event
func handle(ev *fsnotify.FileEvent, done chan bool) {
	command, args := parseArguments(ev)
	cmd := exec.Command(command, args...)

	// Set up pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal("stdout not gotten: ", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal("stderr not gotten: ", err)
	}

	// Clear the screen
	fmt.Print("\033[H\033[2J")

	// Generate the color functions
	yellow := color.New(color.FgYellow, color.Bold).SprintfFunc()
	magenta := color.New(color.FgMagenta, color.Bold).SprintfFunc()
	red := color.New(color.FgRed, color.Bold).SprintfFunc()
	green := color.New(color.FgGreen, color.Bold).SprintfFunc()

	// Print the command in nice colors
	out := fmt.Sprintf("Running %s %s...", yellow(command), magenta(strings.Join(args, " ")))
	log.Println(out)

	// Repeatedly try to start the command.
	// There are cases in which this would fail, and just looping seems to fix it.
	// This is neither nice nor elegant, but hey, it works and if it can be kept
	// simple then it should be.
	for {
		err = cmd.Start()
		if err != nil {
			log.Print("Command failed to start: ", err)
		} else {
			break
		}
	}

	// Set up channels and states for the pipe listening
	outchan := make(chan string)
	errchan := make(chan string)
	outdonechan := make(chan bool)
	errdonechan := make(chan bool)
	outdone := false

	// Trigger the pipes
	go loopOutput(outchan, outdonechan, stdout)
	go loopOutput(errchan, errdonechan, stderr)

	// Handle pipe output and wait for them to drain completely.
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

	done <- true
}

// loopOutput takes a pipe and drains it until EOF.
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
		log.Fatal("Unable to create fsnotify watcher: ", err)
	}

	done := make(chan bool)
	item := make(chan bool)

	// Process events
	go func() {
		running := false
		for {
			select {
			case ev := <-watcher.Event:
				// A watcher event arrived - act on it, but only if
				// no other action is already happening
				if running == false {
					running = true
					go handle(ev, item)
				}

			case <-item:
				// The runner has returned! Let the others roam!
				running = false

			case err := <-watcher.Error:
				log.Println("watcher error: ", err)

			}
		}
	}()

	log.Printf("Watching %s...", *dir)
	err = watcher.Watch(*dir)
	if err != nil {
		log.Fatal("Directory watching failed: ", err)
	}

	<-done
}
