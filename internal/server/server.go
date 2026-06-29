package server

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/vimla/gocache/internal/protocol"
	"github.com/vimla/gocache/internal/store"
)

// Server manages TCP connections and dispatches commands to the store.
type Server struct {
	addr     string
	store    *store.Store
	listener net.Listener
	wg       sync.WaitGroup // tracks active client connections
}

// New creates a new Server bound to the given address.
func New(addr string, s *store.Store) *Server {
	return &Server{
		addr:  addr,
		store: s,
	}
}

// Start begins listening for TCP connections and blocks until ctx is cancelled.
// On shutdown, it closes the listener and waits for active connections to drain.
func (s *Server) Start(ctx context.Context) error {
	var err error
	s.listener, err = net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.addr, err)
	}

	log.Printf("GoCache server listening on %s", s.addr)

	// Run accept loop in a separate goroutine
	go s.acceptLoop(ctx)

	// Block until shutdown signal
	<-ctx.Done()
	log.Println("Shutting down server...")

	// Close listener to unblock Accept()
	s.listener.Close()

	// Wait for active connections to drain, with a timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All connections closed gracefully")
	case <-time.After(10 * time.Second):
		log.Println("Shutdown timed out, forcing exit")
	}

	return nil
}

// acceptLoop continuously accepts new TCP connections until the context
// is cancelled or the listener is closed.
func (s *Server) acceptLoop(ctx context.Context) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			// Check if we're shutting down
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("Accept error: %v", err)
				continue
			}
		}

		s.wg.Add(1)
		go s.handleConnection(ctx, conn)
	}
}

// handleConnection reads commands from a single client connection,
// parses them, executes them against the store, and writes back responses.
func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	log.Printf("Client connected: %s", remoteAddr)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		// Check if server is shutting down
		select {
		case <-ctx.Done():
			conn.Write([]byte("ERROR server shutting down\n"))
			return
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		cmd, err := protocol.Parse(line)
		if err != nil {
			response := fmt.Sprintf("ERROR %s\n", err.Error())
			conn.Write([]byte(response))
			continue
		}

		result := protocol.Execute(cmd, s.store)
		conn.Write([]byte(result + "\n"))
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Client %s read error: %v", remoteAddr, err)
	}

	log.Printf("Client disconnected: %s", remoteAddr)
}
