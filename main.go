package main

import (
	"flag"
	"log"

	"github.com/howeyc/fsnotify"
	"os"
)

func handle(ev *fsnotify.FileEvent) {
	log.Println("event:", ev)
}

func main() {
	dir := flag.String("d", ".", "directory to watch")
	flag.Parse()

	args := os.Args[1:]
	log.Println(args)

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
