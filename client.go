// Client
package main

import (
	"bufio"
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

	// Create a buffered writer
	writer := bufio.NewWriter(conn)

	// Send a few message
	messages := []string{
		"Hello from client!\n",
		"This is line 2\n",
		"And line 3\n",
	}
	for _, msg := range messages {
		_, err = writer.WriteString(msg)
		if err != nil {
			fmt.Println("client: write error:", err)
			return
		}
		writer.Flush() //! Important: flush the buffer to send data
		fmt.Println("client: sent: ", msg)
		time.Sleep(2 * time.Second) // Pause between messages
	}

	// Keep connection open for a bit
	time.Sleep(2 * time.Second)
}
