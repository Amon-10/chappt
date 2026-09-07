package main

import (
	"fmt"
	"net"
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

	// .Write - takes slice of bytes and returns length of the bytes and error object if exists
	// assign returned length of bytes to _(the blank identifier) where it gets thrown away as it is not needed.
	_, err = conn.Write([]byte("hello"))
	if (err != nil) {
		fmt.Println("Error sending messages: ", err)
		return
	}
}