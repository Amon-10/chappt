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

	// make channels for join, leave and broadcast
	join := make(chan net.Conn)
	leave := make(chan net.Conn)
	broadcast := make(chan string)

	// pass channels to manager to handle client behaviour
	go manager(join, leave, broadcast)

	// infinite for loop to accept multiple client connections
	for {
		// Accept incoming connections
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err)
			return
		}

		join <- conn
		go handleConnection(conn, leave, broadcast)
	}
}

// Read messages from client and forward them to the manager
func handleConnection(conn net.Conn, leave chan<- net.Conn, broadcast chan<- string) {
	// notify manager that client left and close connection
	defer func() {
		leave <- conn
		conn.Close()
	}()

	fmt.Println("Client connected")

	scanner := bufio.NewScanner(conn)

	// read new-line delimited messages until the client disconnects
	for scanner.Scan() {
		message := scanner.Text() // gives complete message without trailing \n
		broadcast <- message // send the message through broadcast channel

		fmt.Println("Received:", message)
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("Error during read", err)
	}
}

func manager(join <-chan net.Conn, leave <-chan net.Conn, broadcast <-chan string) {
	clients := make(map[net.Conn]bool) 
	for {
		select {
			case conn := <-join:
				clients[conn] = true
				fmt.Println("client joined", conn)
			
			case conn := <-leave:
				delete(clients, conn)
				fmt.Println("client has left", conn)
			
			case msg := <-broadcast:
				fmt.Println("broadcast requested", msg)
				
				for client := range clients {
					_, err := client.Write([]byte(msg + "\n"))
					if err != nil {
						fmt.Println("Error during broadcasting to client", err)
					}
				}
		}
	}
}
