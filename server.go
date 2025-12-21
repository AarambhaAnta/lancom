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

	// Block forever to keep server running
	select {}
}
