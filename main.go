package main

import (
	"flag"
	"log"

	"github.com/howeyc/fsnotify"
	"os"
	"path/filepath"
)

func handle(ev *fsnotify.FileEvent) {
	log.Println("event:", ev)
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
