package main

import (
	"bufio"
	"fmt"
	"net"
)

func main() {
	// Start a TCP server on port 8080
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error starting server: ", err)
		return
	}
	defer listener.Close()

	fmt.Println("Server listening on :8080")

	// infinite for loop to accept multiple client connections
	for {
		// Accept incoming connections
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err)
			return
		}

		go handleConnection(conn)
	}
}

// Read and display messages
func handleConnection(conn net.Conn) {
	defer conn.Close()

	fmt.Println("Client connected")

	scanner := bufio.NewScanner(conn)

	// allow multiple messages to be recieved
	// scanner.Scan waits until "\n"
	for scanner.Scan() {
		message := scanner.Text() // gives complete message without trailing \n

		fmt.Println("Received:", message)
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("Error during read", err)
	}
}
