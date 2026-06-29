package main

import (
	"flag"
	"fmt"
	"log"
	"os/signal"
	"syscall"
	"time"

	"context"

	"github.com/vimla/gocache/internal/server"
	"github.com/vimla/gocache/internal/store"
)

func main() {
	// Parse command-line flags
	port := flag.Int("port", 6380, "TCP port to listen on")
	flag.Parse()

	// Create a context that cancels on SIGINT (Ctrl+C) or SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initialize the in-memory store
	s := store.New()

	// Start the background TTL reaper (sweeps every 1 second)
	s.StartReaper(ctx, 1*time.Second)

	// Create and start the TCP server
	addr := fmt.Sprintf(":%d", *port)
	srv := server.New(addr, s)

	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[gocache] ")

	if err := srv.Start(ctx); err != nil {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Server stopped.")
}
