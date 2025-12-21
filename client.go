// Client
package main

import (
	"fmt"
	"net"
)

func main() {
	conn, err:= net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		fmt.Println("client: failed to connect")
		panic(err)
	}
	defer conn.Close()

	fmt.Println("client: connected to server")

	// Block forever to keep connection alive
}