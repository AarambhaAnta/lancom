package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		fmt.Println("client: failed to connect")
		panic(err)
	}
	defer conn.Close()

	fmt.Println("client: connected to server")

	reader := bufio.NewReader(os.Stdin)

	// read ( terminal )->write->read ( server )
	for {
		fmt.Print("> ")
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("client: reader failed to read from terminal")
			return
		}

		_, err = conn.Write([]byte(msg))
		if err != nil {
			fmt.Println("client: failed to write")
			return
		}

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println("client: failed to read")
			return
		}

		fmt.Println("server replied: ", string(buf[:n]))
	}
}
