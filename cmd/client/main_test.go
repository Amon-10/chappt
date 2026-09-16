package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/Amon-10/chappt.git/internal/protocol"
)

func TestRegisterRetriesLocalAndServerRejections(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	done := make(chan error, 1)
	go func() {
		reader := bufio.NewReader(server)
		first, err := reader.ReadString('\n')
		if err != nil {
			done <- err
			return
		}
		if first != "Alice\n" {
			done <- fmt.Errorf("first username = %q", first)
			return
		}
		if _, err := fmt.Fprintln(server, protocol.ErrorPrefix+"username is already in use"); err != nil {
			done <- err
			return
		}

		second, err := reader.ReadString('\n')
		if err != nil {
			done <- err
			return
		}
		if second != "Bob\n" {
			done <- fmt.Errorf("second username = %q", second)
			return
		}
		_, err = fmt.Fprintln(server, protocol.Welcome)
		done <- err
	}()

	input := bufio.NewScanner(strings.NewReader("   \nAlice\nBob\n"))
	reader := bufio.NewReader(client)
	var output bytes.Buffer
	if err := register(client, input, reader, &output, terminalPalette(false)); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("server script: %v", err)
	}

	wantMessages := []string{"Username cannot be empty.", "username is already in use"}
	for _, want := range wantMessages {
		if !strings.Contains(output.String(), want) {
			t.Errorf("output %q does not contain %q", output.String(), want)
		}
	}
}

func TestTerminalPaletteCanBeDisabled(t *testing.T) {
	colors := terminalPalette(false)
	if colors != (palette{}) {
		t.Fatalf("disabled palette = %#v, want no escape sequences", colors)
	}
}
