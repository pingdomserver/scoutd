package main

import (
	"log"
	"os/exec"
	"sync"
	"time"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(2)
	go listenForRealtime(&wg)
	go reportLoop(&wg)
	wg.Wait()
}

func reportLoop(wg *sync.WaitGroup) {
	c := time.Tick(1 * time.Second)
	for _ = range c {
		// fmt.Printf("report loop\n")
		cmd := exec.Command("ruby -s http://staging.scoutapp.com oh5inzUpIFFPgBErH719ff0IGLP7vSAhmdeakONI")
		err := cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
	wg.Done()
}

func listenForRealtime(wg *sync.WaitGroup) {
	// add realtime listener
	wg.Done()
}
