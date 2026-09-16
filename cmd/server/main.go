package main

import (
	"bufio"
	"fmt"
	"net"
)

type Client struct {
	name string
	conn net.Conn
}

type Message struct {
	sender net.Conn
	message string
}
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
	join := make(chan Client)
	leave := make(chan net.Conn)
	broadcast := make(chan Message)

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

		go handleConnection(conn, leave, broadcast, join)
	}
}

// Read messages from client and forward them to the manager
func handleConnection(conn net.Conn, leave chan<- net.Conn, broadcast chan<- Message, join chan<- Client) {
	// notify manager that client left and close connection
	defer func() {
		leave <- conn
		conn.Close()
	}()

	fmt.Println("Client connected")

	scanner := bufio.NewScanner(conn)

	if scanner.Scan() {
		username := scanner.Text()
		join <- Client{name: username, conn: conn}
	}

	// read new-line delimited messages until the client disconnects
	for scanner.Scan() {
		message := scanner.Text() // gives complete message without trailing \n
		broadcast <- Message{sender: conn, message: message} // send the message through broadcast channel

	}
	if err := scanner.Err(); err != nil {
		fmt.Println("Error during read", err)
	}
}

func manager(join <-chan Client, leave <-chan net.Conn, broadcast <-chan Message) {
	clients := make(map[net.Conn]Client) 
	for {
		select {
			case client := <-join:
				// add client struct to clients map with conn as key
				clients[client.conn] = client
				fmt.Printf("%v has joined\n", client.name)

				clientName := client.name
				for conn := range clients {
					if conn != client.conn {
						_, err := conn.Write([]byte("# " + clientName + " has joined the chat\n"))
						if err != nil {
							fmt.Println("Error during broadcasting join notification", err)
						}
					}
				}
			
			case leaveConn := <-leave:
				clientName := clients[leaveConn].name
				delete(clients, leaveConn)
				
				fmt.Println("client has left", leaveConn)

				for clientConn := range clients {
					_, err := clientConn.Write([]byte("# " + clientName + " has left the chat\n"))
					if err != nil {
						fmt.Println("Error during broadcasting leave notification")
					}
				}
			
			case msg := <-broadcast:
				client := clients[msg.sender]
				fmt.Printf("%v: %v\n", client.name, msg.message)
				
				for conn := range clients {
					if msg.sender != conn {
						_, err := conn.Write([]byte(client.name + ": " + msg.message + "\n"))
						if err != nil {
							fmt.Println("Error during broadcasting to client", err)
						}
					}
				}
		}
	}
}
