// Command server runs the Chappt TCP chat server.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/Amon-10/chappt.git/internal/protocol"
)

// client represents a registered connection and its display name.
type client struct {
	name string
	conn net.Conn
}

// message is a chat message waiting to be broadcast.
type message struct {
	sender  net.Conn
	content string
}

// joinRequest lets the manager atomically reserve a username and report the
// result to the connection handler.
type joinRequest struct {
	client client
	result chan error
}

// serverConfig contains behavior that can be selected at startup.
type serverConfig struct {
	rejectEmpty bool
}

// chatServer owns the channels used to serialize membership and broadcasts.
// Keeping all client-map access in manager avoids locking around joins,
// disconnects, and duplicate-name checks.
type chatServer struct {
	config    serverConfig
	logger    *log.Logger
	join      chan joinRequest
	leave     chan net.Conn
	broadcast chan message
	done      chan struct{}
}

// newChatServer creates a server with an isolated event manager.
func newChatServer(config serverConfig, logger *log.Logger) *chatServer {
	return &chatServer{
		config:    config,
		logger:    logger,
		join:      make(chan joinRequest),
		leave:     make(chan net.Conn),
		broadcast: make(chan message),
		done:      make(chan struct{}),
	}
}

// handleConnection registers one client, reads newline-delimited messages,
// and reports the disconnect to the manager. A rejected username may be
// replaced without reconnecting.
func (s *chatServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	s.logger.Printf("client connected from %s", conn.RemoteAddr())

	scanner := bufio.NewScanner(conn)
	registered := false

	for !registered && scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		request := joinRequest{
			client: client{name: name, conn: conn},
			result: make(chan error, 1),
		}

		select {
		case s.join <- request:
		case <-s.done:
			return
		}

		select {
		case err := <-request.result:
			if err != nil {
				if !writeLine(conn, protocol.ErrorPrefix+err.Error(), s.logger) {
					return
				}
				continue
			}
			registered = true
			writeLine(conn, protocol.Welcome, s.logger)
		case <-s.done:
			return
		}
	}

	if !registered {
		return
	}

	defer func() {
		select {
		case s.leave <- conn:
		case <-s.done:
		}
	}()

	for scanner.Scan() {
		content := scanner.Text()
		select {
		case s.broadcast <- message{sender: conn, content: content}:
		case <-s.done:
			return
		}
	}

	if err := scanner.Err(); err != nil {
		s.logger.Printf("read from %s: %v", conn.RemoteAddr(), err)
	}
}

// manager serializes all changes to the active-client set and broadcasts
// events. Usernames are compared case-insensitively while preserving the
// spelling chosen by the user for display.
func (s *chatServer) manager() {
	clients := make(map[net.Conn]client)
	names := make(map[string]struct{})

	for {
		select {
		case request := <-s.join:
			nameKey := strings.ToLower(request.client.name)
			var err error
			switch {
			case request.client.name == "":
				err = errors.New("username cannot be empty")
			case hasName(names, nameKey):
				err = errors.New("username is already in use")
			default:
				clients[request.client.conn] = request.client
				names[nameKey] = struct{}{}
			}

			request.result <- err
			if err != nil {
				continue
			}

			s.logger.Printf("%s joined", request.client.name)
			broadcastSystem(clients, request.client.conn, request.client.name+" has joined the chat", s.logger)

		case conn := <-s.leave:
			departing, ok := clients[conn]
			if !ok {
				continue
			}

			delete(clients, conn)
			delete(names, strings.ToLower(departing.name))
			s.logger.Printf("%s left", departing.name)
			broadcastSystem(clients, nil, departing.name+" has left the chat", s.logger)

		case msg := <-s.broadcast:
			sender, ok := clients[msg.sender]
			if !ok {
				continue
			}

			if s.config.rejectEmpty && strings.TrimSpace(msg.content) == "" {
				writeLine(msg.sender, protocol.SystemPrefix+"Empty messages are not allowed", s.logger)
				continue
			}

			s.logger.Printf("%s: %s", sender.name, msg.content)
			for conn := range clients {
				if conn != msg.sender {
					writeLine(conn, sender.name+": "+msg.content, s.logger)
				}
			}

		case <-s.done:
			return
		}
	}
}

// serve accepts connections until the listener is closed.
func (s *chatServer) serve(listener net.Listener) error {
	go s.manager()
	defer close(s.done)

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept connection: %w", err)
		}

		go s.handleConnection(conn)
	}
}

// hasName reports whether the normalized username is already reserved.
func hasName(names map[string]struct{}, name string) bool {
	_, exists := names[name]
	return exists
}

// broadcastSystem sends a status message to every client except exclude.
// Passing nil sends the message to every connected client.
func broadcastSystem(clients map[net.Conn]client, exclude net.Conn, content string, logger *log.Logger) {
	for conn := range clients {
		if conn != exclude {
			writeLine(conn, protocol.SystemPrefix+content, logger)
		}
	}
}

// writeLine writes one protocol frame and records failures in the server log.
func writeLine(conn net.Conn, content string, logger *log.Logger) bool {
	if _, err := fmt.Fprintln(conn, content); err != nil {
		logger.Printf("write to %s: %v", conn.RemoteAddr(), err)
		return false
	}
	return true
}

func main() {
	address := flag.String("addr", ":8080", "TCP address to listen on")
	rejectEmpty := flag.Bool("reject-empty", true, "reject blank or whitespace-only messages")
	flag.Parse()

	logger := log.New(os.Stdout, "chappt: ", log.LstdFlags)
	listener, err := net.Listen("tcp", *address)
	if err != nil {
		logger.Fatalf("start server: %v", err)
	}
	defer listener.Close()

	logger.Printf("listening on %s (reject-empty=%t)", listener.Addr(), *rejectEmpty)
	server := newChatServer(serverConfig{rejectEmpty: *rejectEmpty}, logger)
	if err := server.serve(listener); err != nil {
		logger.Printf("server stopped: %v", err)
	}
}
