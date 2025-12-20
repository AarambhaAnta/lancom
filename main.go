package main

import (
	"fmt"
	"net"
	"sync"
)

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:9000")
	if err != nil {
		fmt.Println("server failed to listen ", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	// client side
	go func() {
		defer wg.Done()
		conn, err := net.Dial("tcp", "127.0.0.1:9000")
		if err != nil {
			fmt.Println("client failed to connect to server", err)
		}

		fmt.Println("client connected")

		// send msg
		_, err = conn.Write([]byte("hello"))
		if err != nil {
			fmt.Println("client: failed to write msg")
		}

		// receive msg
		buf := make([]byte, 64)
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println("client: failed to read msg")
		}

		fmt.Println("client got:", string(buf[:n]))
		conn.Close()
	}()

	// server side
	conn, err := ln.Accept()
	if err != nil {
		fmt.Println("failed to accept")
	}

	fmt.Println("server accepted connection")

	// receive msg
	buf := make([]byte, 64)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println("failed to read msg")
	}

	fmt.Println("server got:", string(buf[:n]))

	// send msg
	_, err = conn.Write([]byte("ok"))
	if err != nil {
		fmt.Println("server: failed to write msg")
	}

	wg.Wait()
	conn.Close()
}
