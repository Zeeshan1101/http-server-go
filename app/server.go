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

type Response struct {
	StatusCode int
	Status     string
	Header     map[string][]string
	Body       string
}

func AcceptForm(res Response) string {
	var response string
	response += fmt.Sprintf("HTTP/1.1 %d %s\r\n", res.StatusCode, res.Status)

	for k, v := range res.Header {
		response += fmt.Sprintf("%s: %s\r\n", k, v[0])
	}
	response += "\r\n"
	response += string(res.Body)

	return response
}

func handleConnection(conn net.Conn) {
	scanner := bufio.NewReader(conn)
	req, err := http.ReadRequest(scanner)

	if err != nil {
		fmt.Fprintln(conn, "reading input", err)
	}

	var response Response

	if req.Method == "GET" {
		switch path := req.URL.Path; {
		case strings.HasPrefix(path, "/echo"):
			suffix := strings.TrimPrefix(path, "/echo/")
			acceptencoding := req.Header.Get("Accept-Encoding")
			if acceptencoding == "gzip" {
				response = Response{
					StatusCode: 200,
					Status:     "OK",
					Header: map[string][]string{
						"Content-Encoding": {"gzip"},
						"Content-Type":     {"text/plain"},
						"Content-Length":   {fmt.Sprintf("%d", len([]byte(suffix)))},
					},
					Body: suffix,
				}
			} else {
				response = Response{
					StatusCode: 200,
					Status:     "OK",
					Header: map[string][]string{
						"Content-Type":   {"text/plain"},
						"Content-Length": {fmt.Sprintf("%d", len([]byte(suffix)))},
					},
					Body: suffix,
				}
			}
		case strings.HasPrefix(path, "/user-agent"):
			useragent := req.Header.Get("User-Agent")
			response = Response{
				StatusCode: 200,
				Status:     "OK",
				Header: map[string][]string{
					"Content-Type":   {"text/plain"},
					"Content-Length": {fmt.Sprintf("%d", len([]byte(useragent)))},
				},
				Body: useragent,
			}
		case strings.HasPrefix(path, "/files"):
			files := strings.TrimPrefix(path, "/files/")
			file, err := os.ReadFile(filedir + files)
			if err != nil {
				response = generateResponse(404, "Not Found")
				break
			}
			response = Response{
				StatusCode: 200,
				Status:     "OK",
				Header: map[string][]string{
					"Content-Type":   {"application/octet-stream"},
					"Content-Length": {fmt.Sprintf("%d", len([]byte(file)))},
				},
				Body: string(file),
			}
		case path == "/":
			response = generateResponse(200, "OK")
		default:
			response = generateResponse(404, "Not Found")
		}
	}

	if req.Method == "POST" {
		switch path := req.URL.Path; {
		case strings.HasPrefix(path, "/files/"):
			files := strings.TrimPrefix(path, "/files/")
			data, err := io.ReadAll(req.Body)
			if err != nil {
				response = generateResponse(404, "File Cannot Be Read")
				break
			}
			defer req.Body.Close()
			err = os.WriteFile(filedir+files, []byte(data), 0644)
			if err != nil {
				response = generateResponse(404, "File Cannot Be Written")
				break
			}
			response = generateResponse(201, "Created")
		default:
			response = generateResponse(404, "Not Found")
		}
	}

	conn.Write([]byte(AcceptForm(response)))
	conn.Close()

}

func generateResponse(statusCode int, statusText string) Response {
	return Response{
		StatusCode: statusCode,
		Status:     statusText,
	}
}
