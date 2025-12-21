// Client
package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		fmt.Println("client: failed to connect")
		panic(err)
	}
	defer conn.Close()

	fmt.Println("client: connected to server")

	// Send a message
	message := "Hello from client!\n"
	_, err = conn.Write([]byte(message))
	if err != nil {
		fmt.Println("client: write error:", err)
		return
	}

	fmt.Println("client: sent message:", message)

	// Keep connection open for a bit
	time.Sleep(2 * time.Second)
}
