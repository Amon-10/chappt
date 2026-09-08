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

	fmt.Println("Enter message(Press ctrl+c to exit): ")

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