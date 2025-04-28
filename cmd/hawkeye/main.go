package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nemuizzz/hawkeye/cmd/hawkeye/commands"
)

func main() {
	// Set up signal handling for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nShutting down...")
		os.Exit(0)
	}()

	// Execute the root command
	if err := commands.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
