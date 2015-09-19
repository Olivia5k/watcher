package main

import (
	"log"

	"github.com/howeyc/fsnotify"
)

func handle(ev *fsnotify.FileEvent) {
	log.Println("event:", ev)
}

func main() {
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

	err = watcher.Watch("testDir")
	if err != nil {
		log.Fatal(err)
	}

	<-done

}
