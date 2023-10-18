package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/ipfs-shipyard/nopfs"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "Usage: program <local_denylist> <source_URL>")
		os.Exit(1)
	}

	local := os.Args[1]
	remote := os.Args[2]

	fmt.Printf("%s: subscribed to %s. CTRL-C to stop\n", local, remote)

	subscriber, err := nopfs.NewHTTPSubscriber(remote, local, 1*time.Minute)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	fmt.Println("Stopping")
	subscriber.Stop()
}
