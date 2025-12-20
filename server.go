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

	fmt.Println("server: listening...")

	conn, err := ln.Accept()
	if err != nil {
		fmt.Println("server: failed to reconnect")
		panic(err)
	}
	defer conn.Close()

	fmt.Println("server: client connected...")

	// read->print->write
	reader := bufio.NewReader(conn)

	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("server: failed to read")
			return
		}

		fmt.Println("server got: ", msg)

		_, err = conn.Write([]byte("ok\n"))
		if err != nil {
			fmt.Println("server: failed to write")
			return
		}
	}
}
