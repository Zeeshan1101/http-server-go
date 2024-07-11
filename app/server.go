package main

import (
	"flag"
	"fmt"
	"regexp"
	"strings"

	"net"
	"os"
)

var filedir string

func init() {
	flag.StringVar(&filedir, "directory", "", "dir")
}

type RouteHandler func(req Request, res Response) Response

type Route struct {
	handler RouteHandler
	regex   string
	key     []string
	Method  Method
}

type Headers map[string][]string

func (h Headers) Get(key string) string {

	if h[key] == nil {
		return ""
	}

	if len(h[key]) == 1 {
		return h[key][0]

	}

	if key == "Accept-Encoding" {
		fmt.Println(h[key])
		for _, v := range h[key] {
			fmt.Println(v)
			if v == "gzip" {
				return "gzip"
			}
		}
	}
	return strings.Join(h[key], ",")
}

type HttpRouter struct {
	Host     string
	Port     string
	Protocol string
	Routes   map[string]Route
}

func (r *HttpRouter) Run() {

	l, err := net.Listen(r.Protocol, r.Host+":"+r.Port)
	if err != nil {
		fmt.Println("Failed to bind to port ", r.Port)
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		defer conn.Close()
		go r.handleConnection(conn)
	}
}

func PathRegex(path string) string {
	path = regexp.MustCompile(`\/`).ReplaceAllString(path, `\/`)
	path = regexp.MustCompile(`:(\w+)`).ReplaceAllString(path, `([^\/]+)`)
	path = regexp.MustCompile(`\*`).ReplaceAllString(path, `.*`)
	return `^` + path + `$`
}

func GetParams(path string) []string {
	params := []string{}
	find := regexp.MustCompile(`:(\w+)`)
	split := find.FindAllString(path, -1)
	for _, v := range split {
		params = append(params, strings.Split(v, ":")[1])
	}
	return params
}

func (r *HttpRouter) AddRoute(method Method, path string, handler RouteHandler) {

	key := GetParams(path)
	path = PathRegex(path)
	r.Routes[method.String()+path] = Route{
		handler: handler,
		regex:   path,
		key:     key,
		Method:  method,
	}
	return
}

func (r *HttpRouter) HandlerIncoming(conn net.Conn) Response {
	req := ParseRequest(conn)
	routename := req.Method.String() + req.Path
	route, exists := r.Routes[routename]
	if !exists {
		for _, v := range r.Routes {
			if v.Method != req.Method {
				continue
			}
			reg := regexp.MustCompile(v.regex)
			if reg.Match([]byte(req.Path)) {
				route = v
				fmt.Println(route)
				break
			}
		}
	}

	if route.handler == nil {
		return Response{
			StatusCode: 404,
			Status:     "Not Found",
		}
	}

	if len(route.key) > 0 {
		path := req.Path
		reg := regexp.MustCompile(route.regex)
		match := reg.FindAllStringSubmatch(path, 1)
		for i, k := range match[0][1:] {
			req.Params[route.key[i]] = k
			path = strings.Replace(path, match[0][i+1]+"/", "", 1)
		}
	}

	res := Response{
		StatusCode: 200,
		Status:     "OK",
	}

	return route.handler(req, res)

}

type Method int

const (
	GET Method = iota + 1
	POST
	PUT
	DELETE
	NOT
)

func (m Method) String() string {
	return [...]string{"GET", "POST", "PUT", "DELETE"}[m-1]
}

func ReadString(method string) Method {
	switch method {
	case "GET":
		return GET
	case "POST":
		return POST
	case "PUT":
		return PUT
	case "DELETE":
		return DELETE
	default:
		return NOT
	}
}

type Request struct {
	Method      Method
	Path        string
	Headers     Headers
	Body        string
	HTTPVersion string
	Params      map[string]string
}

func ParseRequest(conn net.Conn) Request {
	buf := make([]byte, 1024)
	conn.Read(buf)
	lines := strings.Split(string(buf), "\r\n")
	request := Request{
		Headers: make(map[string][]string),
		Params:  make(map[string]string),
	}

	for i := 0; i <= len(lines); i++ {
		if i == 0 {
			line := strings.Split(lines[i], " ")
			method := ReadString(line[0])
			if method == NOT {
				break
			}
			request.Method = method
			request.Path = line[1]
			request.HTTPVersion = line[2]
			continue
		}

		headers := strings.Split(lines[i], ": ")

		if len(headers) < 2 {
			request.Body = strings.TrimSpace(strings.ReplaceAll(lines[i+1], "\x00", ""))
			break
		}

		var headersKey []string
		for _, v := range strings.Split(headers[1], ",") {
			headersKey = append(headersKey, strings.TrimSpace(v))
		}

		request.Headers[headers[0]] = headersKey
	}
	return request
}

type Response struct {
	StatusCode int
	Status     string
	Headers    Headers
	Body       string
}

func (res *Response) WriteResponse() string {
	var response string
	response += fmt.Sprintf("HTTP/1.1 %d %s\r\n", res.StatusCode, res.Status)

	for k, v := range res.Headers {
		response += fmt.Sprintf("%s: %s\r\n", k, v[0])
	}

	response += "\r\n"
	response += string(res.Body)

	return response
}

func main() {
	flag.Parse()
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	fmt.Println(filedir)

	router := HttpRouter{
		Protocol: "tcp",
		Port:     "4221",
		Host:     "0.0.0.0",
		Routes:   make(map[string]Route),
	}

	router.AddRoute(GET, "/", func(req Request, res Response) Response {
		return res
	})

	router.AddRoute(GET, "/echo/:suffix", func(req Request, res Response) Response {
		suffix := req.Params["suffix"]
		acceptencoding := req.Headers.Get("Accept-Encoding")
		fmt.Println(acceptencoding)
		if acceptencoding == "gzip" {
			res.Headers = map[string][]string{
				"Content-Encoding": {"gzip"},
				"Content-Type":     {"text/plain"},
				"Content-Length":   {fmt.Sprintf("%d", len([]byte(suffix)))},
			}
			res.Body = suffix
		} else {
			res.Headers = map[string][]string{
				"Content-Type":   {"text/plain"},
				"Content-Length": {fmt.Sprintf("%d", len([]byte(suffix)))},
			}
			res.Body = suffix
		}
		return res
	})

	router.AddRoute(GET, "/user-agent", func(req Request, res Response) Response {
		useragent := req.Headers["User-Agent"][0]
		res.Headers = map[string][]string{
			"Content-Type":   {"text/plain"},
			"Content-Length": {fmt.Sprintf("%d", len([]byte(useragent)))},
		}
		res.Body = useragent
		return res
	})

	router.AddRoute(GET, "/files/:id", func(req Request, res Response) Response {
		files := req.Params["id"]
		file, err := os.ReadFile(filedir + files)
		if err != nil {
			return Response{
				StatusCode: 404,
				Status:     "Not Found",
			}
		}

		res.Headers = map[string][]string{
			"Content-Type":   {"application/octet-stream"},
			"Content-Length": {fmt.Sprintf("%d", len([]byte(file)))},
		}
		res.Body = string(file)
		return res
	})

	router.AddRoute(POST, "/files/:name", func(req Request, res Response) Response {
		name := req.Params["name"]
		err := os.WriteFile(filedir+name, []byte(req.Body), 0644)
		if err != nil {
			return Response{
				StatusCode: 404,
				Status:     "Not Found",
			}
		}
		return Response{
			StatusCode: 201,
			Status:     "Created",
		}
	})

	router.Run()
}

func (r *HttpRouter) handleConnection(conn net.Conn) {

	res := r.HandlerIncoming(conn)
	fmt.Println(res.WriteResponse())

	conn.Write([]byte(res.WriteResponse()))

	conn.Close()
}
