package main

import (
	"fmt"
	"net"
)

func main(){
	// Start a TCP server on port 8080
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error starting server: ", err)
		return
	}
	defer listener.Close()
	
	fmt.Println("Server listening on :8080")

	// Accept incoming connections
	conn, err := listener.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err)
		return
	}
	defer conn.Close()

	fmt.Println("Client connected")
	
	buffer := make([]byte, 1024)

	for {
		n, err := conn.Read(buffer)
		if (err != nil) {
			fmt.Println("Error reading message: ", err)
			return
		}
		fmt.Println("Received:", string(buffer[:n]))
	}
}