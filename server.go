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

	// Block forever to keep connection alive
	select {}
}
