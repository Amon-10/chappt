package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/Amon-10/chappt.git/internal/protocol"
)

const testTimeout = 2 * time.Second

// testClient wraps a real TCP connection with line-oriented test helpers.
type testClient struct {
	conn   net.Conn
	reader *bufio.Reader
}

// startTestServer listens on a random local port and returns its cleanup hook.
func startTestServer(t *testing.T, rejectEmpty bool) (string, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	server := newChatServer(
		serverConfig{rejectEmpty: rejectEmpty},
		log.New(io.Discard, "", 0),
	)
	done := make(chan error, 1)
	go func() {
		done <- server.serve(listener)
	}()

	cleanup := func() {
		if err := listener.Close(); err != nil && !strings.Contains(err.Error(), "closed network connection") {
			t.Errorf("close listener: %v", err)
		}
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("serve: %v", err)
			}
		case <-time.After(testTimeout):
			t.Error("server did not stop")
		}
	}

	return listener.Addr().String(), cleanup
}

// dialTestClient opens a real TCP connection to the test server.
func dialTestClient(t *testing.T, address string) *testClient {
	t.Helper()

	conn, err := net.DialTimeout("tcp", address, testTimeout)
	if err != nil {
		t.Fatalf("dial %s: %v", address, err)
	}
	return &testClient{conn: conn, reader: bufio.NewReader(conn)}
}

// send writes one complete protocol frame.
func (c *testClient) send(t *testing.T, line string) {
	t.Helper()
	c.conn.SetWriteDeadline(time.Now().Add(testTimeout))
	if _, err := fmt.Fprintln(c.conn, line); err != nil {
		t.Fatalf("send %q: %v", line, err)
	}
}

// read returns one complete protocol frame without its newline.
func (c *testClient) read(t *testing.T) string {
	t.Helper()
	c.conn.SetReadDeadline(time.Now().Add(testTimeout))
	line, err := c.reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return strings.TrimSuffix(line, "\n")
}

// register sends a username and verifies that it was accepted.
func (c *testClient) register(t *testing.T, name string) {
	t.Helper()
	c.send(t, name)
	if got := c.read(t); got != protocol.Welcome {
		t.Fatalf("register %q: got %q, want %q", name, got, protocol.Welcome)
	}
}

// expect verifies the next frame received by the client.
func (c *testClient) expect(t *testing.T, want string) {
	t.Helper()
	if got := c.read(t); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDuplicateUsernameCanRetryWithoutReconnect(t *testing.T) {
	address, stop := startTestServer(t, true)
	defer stop()

	alice := dialTestClient(t, address)
	defer alice.conn.Close()
	alice.register(t, "Alice")

	duplicate := dialTestClient(t, address)
	defer duplicate.conn.Close()
	duplicate.send(t, "  aLiCe  ")
	duplicate.expect(t, protocol.ErrorPrefix+"username is already in use")

	duplicate.register(t, "Bob")
	alice.expect(t, protocol.SystemPrefix+"Bob has joined the chat")

	duplicate.send(t, "hello after retry")
	alice.expect(t, "Bob: hello after retry")
}

func TestEmptyUsernameIsRejectedByServer(t *testing.T) {
	address, stop := startTestServer(t, true)
	defer stop()

	client := dialTestClient(t, address)
	defer client.conn.Close()
	client.send(t, "   ")
	client.expect(t, protocol.ErrorPrefix+"username cannot be empty")
	client.register(t, "Alice")
}

func TestEmptyMessagePolicy(t *testing.T) {
	t.Run("rejects whitespace when enabled", func(t *testing.T) {
		address, stop := startTestServer(t, true)
		defer stop()

		alice := dialTestClient(t, address)
		defer alice.conn.Close()
		alice.register(t, "Alice")
		bob := dialTestClient(t, address)
		defer bob.conn.Close()
		bob.register(t, "Bob")
		alice.expect(t, protocol.SystemPrefix+"Bob has joined the chat")

		alice.send(t, "   ")
		alice.expect(t, protocol.SystemPrefix+"Empty messages are not allowed")

		bob.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		if line, err := bob.reader.ReadString('\n'); err == nil {
			t.Fatalf("Bob unexpectedly received %q", line)
		}
	})

	t.Run("broadcasts whitespace when disabled", func(t *testing.T) {
		address, stop := startTestServer(t, false)
		defer stop()

		alice := dialTestClient(t, address)
		defer alice.conn.Close()
		alice.register(t, "Alice")
		bob := dialTestClient(t, address)
		defer bob.conn.Close()
		bob.register(t, "Bob")
		alice.expect(t, protocol.SystemPrefix+"Bob has joined the chat")

		alice.send(t, "   ")
		bob.expect(t, "Alice:    ")
	})
}

func TestThreeClientStressJoinsLeavesMessagesAndReconnects(t *testing.T) {
	address, stop := startTestServer(t, true)
	defer stop()

	alice := dialTestClient(t, address)
	defer alice.conn.Close()
	alice.register(t, "Alice")

	bob := dialTestClient(t, address)
	defer bob.conn.Close()
	bob.register(t, "Bob")
	alice.expect(t, protocol.SystemPrefix+"Bob has joined the chat")

	cara := dialTestClient(t, address)
	cara.register(t, "Cara")
	alice.expect(t, protocol.SystemPrefix+"Cara has joined the chat")
	bob.expect(t, protocol.SystemPrefix+"Cara has joined the chat")

	clients := []*testClient{alice, bob, cara}
	names := []string{"Alice", "Bob", "Cara"}
	for round := 0; round < 25; round++ {
		for senderIndex, sender := range clients {
			content := fmt.Sprintf("round-%02d-from-%s", round, names[senderIndex])
			sender.send(t, content)
			for receiverIndex, receiver := range clients {
				if receiverIndex != senderIndex {
					receiver.expect(t, names[senderIndex]+": "+content)
				}
			}
		}
	}

	for reconnect := 0; reconnect < 3; reconnect++ {
		if err := cara.conn.Close(); err != nil {
			t.Fatalf("disconnect Cara: %v", err)
		}
		alice.expect(t, protocol.SystemPrefix+"Cara has left the chat")
		bob.expect(t, protocol.SystemPrefix+"Cara has left the chat")

		cara = dialTestClient(t, address)
		cara.register(t, "Cara")
		alice.expect(t, protocol.SystemPrefix+"Cara has joined the chat")
		bob.expect(t, protocol.SystemPrefix+"Cara has joined the chat")

		content := fmt.Sprintf("reconnected-%d", reconnect)
		cara.send(t, content)
		alice.expect(t, "Cara: "+content)
		bob.expect(t, "Cara: "+content)
	}
	defer cara.conn.Close()
}
