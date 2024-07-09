package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"

	"net"
	"os"
)

var filedir string

func init() {
	flag.StringVar(&filedir, "directory", "", "dir")
}

func main() {
	flag.Parse()
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	fmt.Println(filedir)
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
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	scanner := bufio.NewReader(conn)
	req, err := http.ReadRequest(scanner)

	if err != nil {
		fmt.Fprintln(conn, "reading input", err)
	}

	var response string

	if req.Method == "GET" {
		switch path := req.URL.Path; {
		case strings.HasPrefix(path, "/echo"):
			suffix := strings.TrimPrefix(path, "/echo/")
			response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len([]byte(suffix)), suffix)
		case strings.HasPrefix(path, "/user-agent"):
			useragent := req.Header.Get("User-Agent")
			response = fmt.Sprintf("%s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", generateResponse(200, "OK"), len([]byte(useragent)), useragent)
		case strings.HasPrefix(path, "/files"):
			files := strings.TrimPrefix(path, "/files/")
			file, err := os.ReadFile(filedir + files)
			if err != nil {
				response = generateResponse(404, "Not Found") + "\r\n\r\n"
				break
			}
			response = fmt.Sprintf("%s\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", generateResponse(200, "OK"), len(file), file)
		case path == "/":
			response = generateResponse(200, "OK") + "\r\n\r\n"
		default:
			response = generateResponse(404, "Not Found") + "\r\n\r\n"
		}
	}

	if req.Method == "POST" {
		switch path := req.URL.Path; {
		case strings.HasPrefix(path, "/files/"):
			files := strings.TrimPrefix(path, "/files/")
			data, err := io.ReadAll(req.Body)
			if err != nil {
				response = generateResponse(404, "File Cannot Be Read") + "\r\n\r\n"
				break
			}
			defer req.Body.Close()
			err = os.WriteFile(filedir+files, []byte(data), 0644)
			if err != nil {
				response = generateResponse(404, "File Cannot Be Written") + "\r\n\r\n"
				break
			}
			response = generateResponse(201, "Created") + "\r\n\r\n"
		default:
			response = generateResponse(404, "Not Found") + "\r\n\r\n"
		}
	}

	conn.Write([]byte(response))
	conn.Close()

}

func generateResponse(statusCode int, statusText string) string {
	return fmt.Sprintf("HTTP/1.1 %d %s", statusCode, statusText)
}
