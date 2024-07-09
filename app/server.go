package main

import (
	"fmt"
	"strings"
	// Uncomment this block to pass the first stage
	"net"
	"os"
)

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	// Uncomment this block to pass the first stage
	//
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		defer conn.Close()

		buf := make([]byte, 1024)
		_, err = conn.Read(buf)
		if err != nil {
			conn.Write([]byte("Could Resolve Your Request"))
		}
		reqarr := strings.Split(string(buf), " ")

		url := reqarr[1]
		if strings.HasPrefix("/echo", url) {
			conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		} else if strings.HasPrefix(url, "/echo") {
			suffix := strings.TrimPrefix(url, "/echo/")
			conn.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len([]byte(suffix)), suffix)))
		} else {
			conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		}

	}
}
