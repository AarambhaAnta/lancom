package main

import (
	"bufio"
	"fmt"
	"net"
)

func handleClient(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("server: client disconnected")
			return
		}

		fmt.Println("server got: ", msg)

		_, err = conn.Write([]byte("ok\n"))
		if err != nil {
			return
		}
	}
}

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:9000")
	if err != nil {
		fmt.Println("server: failed to start")
		panic(err)
	}
	defer ln.Close()

	fmt.Println("server: listening...")

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("server: accept error->", err)
			continue
		}

		fmt.Println("server: client connected...")
		go func() {
			handleClient(conn)
		}()

	}

}
