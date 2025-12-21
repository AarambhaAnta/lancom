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

	// Create readers and writers
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// Send a few message and read response
	messages := []string{
		"Hello from client!\n",
		"This is line 2\n",
		"And line 3\n",
	}
	
	for _, msg := range messages {
		// Send message
		_, err = writer.WriteString(msg)
		if err != nil {
			fmt.Println("client: write error:", err)
			return
		}
		writer.Flush()
		fmt.Println("client: sent:", msg)

		// Read response
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("client: read error:", err)
			return
		}
		fmt.Println("client: received: ", response)

		// Pause between messages
		time.Sleep(1 * time.Second)
	}

	// Keep connection open for a bit
	time.Sleep(2 * time.Second)
}
