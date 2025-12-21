// server
package main

import (
	"bufio"
	"fmt"
	"net"
)

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:9000")
	if err != nil {
		fmt.Println("server: failed to start")
		panic(err)
	}
	defer ln.Close()

	fmt.Println("server: listening on 127.0.0.1:9000")

	// Accept a single connection
	conn, err := ln.Accept()
	if err != nil {
		fmt.Println("server: accept error:", err)
		return
	}
	defer conn.Close()

	fmt.Println("server: client connected from,", conn.RemoteAddr())

	// Create a buffered reader
	reader := bufio.NewReader(conn)

	// Read lines in a loop
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("server: read error:", err)
			return
		}

		fmt.Println("server: received: ", message)
	}
}
