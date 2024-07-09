package main

import (
	"bufio"
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

		scanner := bufio.NewScanner(conn)

		req, err := ParseRequest(scanner)

		if err != nil {
			fmt.Fprintln(conn, "reading input", err)
		}

		var response string

		switch path := req.Path; {
		case strings.HasPrefix(path, "/echo"):
			suffix := strings.TrimPrefix(path, "/echo/")
			response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len([]byte(suffix)), suffix)
		case strings.HasPrefix(path, "/user-agent"):
			response = fmt.Sprintf("%s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", generateResponse(200, "OK"), len([]byte(req.UserAgent)), req.UserAgent)
		case path == "/":
			response = generateResponse(200, "OK") + "\r\n\r\n"
		default:
			response = generateResponse(404, "Not Found") + "\r\n\r\n"
		}

		conn.Write([]byte(response))
	}
}
func generateResponse(statusCode int, statusText string) string {
	return fmt.Sprintf("HTTP/1.1 %d %s", statusCode, statusText)
}

type HttpRequest struct {
	Path      string
	Method    string
	Headers   map[string]string
	Body      string
	UserAgent string
}

func ParseRequest(scanner *bufio.Scanner) (*HttpRequest, error) {
	var options HttpRequest = HttpRequest{}
	options.Headers = make(map[string]string)

	for i := 0; scanner.Scan(); i++ {
		if i == 0 {
			parts := strings.Split(scanner.Text(), " ")
			options.Method = parts[0]
			options.Path = parts[1]
			continue
		}
		headers := strings.Split(scanner.Text(), ": ")

		if len(headers) < 2 {
			options.Body = headers[0]
			break
		}

		if headers[0] == "User-Agent" {
			options.UserAgent = headers[1]
		}

		options.Headers[headers[0]] = headers[1]
	}

	return &options, nil
}
