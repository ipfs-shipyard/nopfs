package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ipfs-shipyard/nopfs"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
)

func printPrompt() {
	fmt.Print("> ")
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("> c <cid>")
	fmt.Println("> p <path>")
}

func main() {
	logging.SetLogLevel("nopfs", "DEBUG")
	filename := "test.deny"
	if len(os.Args) < 2 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		fmt.Println("nopfs: denylist testing REPL")
		fmt.Println()
		fmt.Println("Usage: ./nopfs list.deny")
		return
	}
	filename = os.Args[1]
	blocker, err := nopfs.NewBlocker([]string{filename})
	if err != nil {
		fmt.Println(err)
		return
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)

	reader := bufio.NewScanner(os.Stdin)
	printUsage()
	fmt.Println("[CTRL-D to exit]")
	printPrompt()
	for reader.Scan() {
		select {
		case <-sigCh:
			fmt.Println(blocker.Close())
			return
		default:
		}

		text := reader.Text()
		typ, elem, found := strings.Cut(text, " ")
		if !found {
			fmt.Println("not found")
			printPrompt()
			continue
		}
		switch typ {
		case "p":
			status := blocker.IsPathBlocked(path.FromString(elem))
			fmt.Printf("%s: %s\n", status.Status, status.Entry)
		case "c":
			c, err := cid.Decode(elem)
			if err != nil {
				fmt.Println(err)
			} else {
				status := blocker.IsCidBlocked(c)
				fmt.Println(status)
			}
		default:
			fmt.Println("Usage:")
			fmt.Println("> c <cid>")
			fmt.Println("> p <path>")
		}
		printPrompt()

	}
	fmt.Println()
	fmt.Println(blocker.Close())
}
