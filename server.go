// server
package main

import (
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

	// Read data in a loop
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println("server: read error:", err)
			return
		}

		message := string(buf[:n])
		fmt.Printf("server: received %d bytes: %s", n, message)
	}
}
