// Command client runs the interactive Chappt terminal client.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/Amon-10/chappt.git/internal/protocol"
)

// palette stores terminal escape sequences. They are empty when color output
// is disabled, which keeps redirected output and NO_COLOR environments clean.
type palette struct {
	accent string
	dim    string
	green  string
	yellow string
	reset  string
}

// terminalPalette chooses a restrained color scheme for interactive output.
func terminalPalette(enabled bool) palette {
	if !enabled {
		return palette{}
	}
	return palette{
		accent: "\033[1;36m",
		dim:    "\033[2m",
		green:  "\033[32m",
		yellow: "\033[33m",
		reset:  "\033[0m",
	}
}

// useColor reports whether stdout is an interactive terminal and color has
// not been disabled by flag or by the NO_COLOR convention.
func useColor(disabled bool) bool {
	if disabled || os.Getenv("NO_COLOR") != "" {
		return false
	}
	info, err := os.Stdout.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// printBanner displays a compact connection header.
func printBanner(out io.Writer, colors palette, address string) {
	fmt.Fprintf(out, "%s┌──────────────────────────────┐%s\n", colors.accent, colors.reset)
	fmt.Fprintf(out, "%s│            CHAPPT            │%s\n", colors.accent, colors.reset)
	fmt.Fprintf(out, "%s└──────────────────────────────┘%s\n", colors.accent, colors.reset)
	fmt.Fprintf(out, "%sConnected to %s%s\n\n", colors.green, address, colors.reset)
}

// register prompts until the server accepts a non-empty, unique username.
// The server response makes duplicate-name handling explicit and lets the user
// retry without opening a new TCP connection.
func register(conn net.Conn, input *bufio.Scanner, server *bufio.Reader, out io.Writer, colors palette) error {
	for {
		fmt.Fprintf(out, "%sUsername%s: ", colors.accent, colors.reset)
		if !input.Scan() {
			if err := input.Err(); err != nil {
				return fmt.Errorf("read username: %w", err)
			}
			return io.EOF
		}

		username := strings.TrimSpace(input.Text())
		if username == "" {
			fmt.Fprintf(out, "%sUsername cannot be empty.%s\n", colors.yellow, colors.reset)
			continue
		}

		if _, err := fmt.Fprintln(conn, username); err != nil {
			return fmt.Errorf("send username: %w", err)
		}

		response, err := server.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read registration response: %w", err)
		}
		response = strings.TrimSpace(response)

		switch {
		case response == protocol.Welcome:
			return nil
		case strings.HasPrefix(response, protocol.ErrorPrefix):
			fmt.Fprintf(out, "%s%s%s\n", colors.yellow, strings.TrimPrefix(response, protocol.ErrorPrefix), colors.reset)
		default:
			return fmt.Errorf("unexpected registration response %q", response)
		}
	}
}

// receive prints newline-delimited server messages until the connection ends.
func receive(server *bufio.Reader, out io.Writer, colors palette) error {
	for {
		line, err := server.ReadString('\n')
		if line != "" {
			line = strings.TrimSuffix(line, "\n")
			if strings.HasPrefix(line, protocol.SystemPrefix) {
				fmt.Fprintf(out, "\r%s%s%s\n> ", colors.dim, line, colors.reset)
			} else {
				fmt.Fprintf(out, "\r%s\n> ", line)
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("receive message: %w", err)
		}
	}
}

func main() {
	address := flag.String("addr", "localhost:8080", "chat server address")
	noColor := flag.Bool("no-color", false, "disable ANSI colors")
	flag.Parse()

	conn, err := net.Dial("tcp", *address)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to %s: %v\n", *address, err)
		return
	}
	defer conn.Close()

	colors := terminalPalette(useColor(*noColor))
	printBanner(os.Stdout, colors, *address)

	input := bufio.NewScanner(os.Stdin)
	server := bufio.NewReader(conn)
	if err := register(conn, input, server, os.Stdout, colors); err != nil {
		if !errors.Is(err, io.EOF) {
			fmt.Fprintf(os.Stderr, "Registration failed: %v\n", err)
		}
		return
	}

	fmt.Fprintf(os.Stdout, "\n%sType a message and press Enter. Press Ctrl+D to leave.%s\n> ", colors.dim, colors.reset)
	go func() {
		if err := receive(server, os.Stdout, colors); err != nil {
			fmt.Fprintf(os.Stderr, "\nConnection ended: %v\n", err)
		}
	}()

	for input.Scan() {
		if _, err := fmt.Fprintln(conn, input.Text()); err != nil {
			fmt.Fprintf(os.Stderr, "\nUnable to send message: %v\n", err)
			return
		}
		fmt.Fprint(os.Stdout, "> ")
	}

	if err := input.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "\nUnable to read input: %v\n", err)
	}
}
