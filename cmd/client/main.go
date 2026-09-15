package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main(){
	// Initiate connection to listening tcp server
	conn, err := net.Dial("tcp", "localhost:8080")
	if (err != nil) {
		fmt.Println("Error connecting to server: ", err)
		return
	}
	defer conn.Close() // defer - run when surrounding function returns

	fmt.Println("Connected to server")

	// Initialize new scanner
	scanner := bufio.NewScanner(os.Stdin)

	// Get username
	fmt.Println("Enter username: ")
	if scanner.Scan() {
		username := scanner.Text() + "\n"

		_, err := conn.Write([]byte(username))
		if err != nil {
			fmt.Println("Error when setting username", err)
			return
		}
	}

	fmt.Println("Enter message(Press ctrl+c to exit): ")

	// Present incoming broadcast messages
	// pass conn to incomingMsg
	go incomingMsg(conn)

	// Loop and await terminal input
	for scanner.Scan(){
		message := scanner.Text() + "\n"
		
		// .Write - takes slice of bytes and returns length of the bytes and error object if exists
		// assign returned length of bytes to _(the blank identifier) where it gets thrown away as it is not needed.
		_, err := conn.Write([]byte(message))
		if err != nil {
			fmt.Println("Error sending messages: ", err)
			break
		}
	}

	// Check for any terminal scanning errors
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading standard input: ", err)
	}
	
}

// read incoming broadcast messages and print to terminal
func incomingMsg(conn net.Conn) {
	broadcastScanner := bufio.NewScanner(conn)

	for broadcastScanner.Scan() {
		broadcastMsg := broadcastScanner.Text()
		fmt.Printf("%v\n", broadcastMsg)
	}
	if err := broadcastScanner.Err(); err != nil {
		fmt.Println("Error during receiving broadcast", err)
	}
}