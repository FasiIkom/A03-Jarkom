package main

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"net"
	"strings"
)

const (
	SERVER_HOST  = "0.0.0.0"
	SERVER_PORT  = "7481"
	SERVER_TYPE  = "tcp"
	BUFFER_SIZE  = 2048
	STUDENT_NAME = "Firaz"
	STUDENT_NPM  = "2306217481"
	CRLF         = "\r\n"
)

type Student struct {
	Nama string
	Npm  string
}

type GreetResponse struct {
	Student Student
	Greeter string
}

type HttpRequest struct {
	Method         string
	Uri            string
	Version        string
	Host           string
	Accept         string
	AcceptEncoding string
}

type HttpResponse struct {
	Version         string
	StatusCode      string
	ContentType     string
	ContentEncoding string
	ContentLength   int
	Data            []byte
}

func main() {
	listener, err := net.Listen(SERVER_TYPE, SERVER_HOST+":"+SERVER_PORT)
	if err != nil {
		fmt.Println("Error occured")
		return
	}
	defer listener.Close()
	for {
        connection, err := listener.Accept()
        if err != nil {
            fmt.Printf("Error accepting connection: %v\n", err)
            continue
        }
        go HandleConnection(connection)
    }
}

func HandleConnection(connection net.Conn) {
	defer connection.Close()

	buffer := make([]byte, BUFFER_SIZE)
	_, err := connection.Read(buffer)
	if err != nil {
		fmt.Printf("Error reading data: %v\n", err)
		return
	}
	request := RequestDecoder(buffer)
	response := HandleRequest(request)
	responseBytes := ResponseEncoder(response)
    _, err = connection.Write(responseBytes)
    if err != nil {
        fmt.Printf("Error sending response: %v\n", err)
    }
}

func HandleRequest(req HttpRequest) HttpResponse {
	var response HttpResponse
	response.Version = "HTTP/1.1"

	if req.Uri == "/" {
		response.StatusCode = "200"
		response.ContentType = "text/html"
		response.Data = []byte("<html><body><h1>Halo, dunia! Aku Firaz sedang mengerjakan A03</h1></body></html>")
		response = applyEncoding(req, response)
		return response
	}

	if strings.HasPrefix(req.Uri, "/greet/") {
		parts := strings.SplitN(req.Uri, "?", 2)
		path := parts[0]
		query := ""
		if len(parts) == 2 {
			query = parts[1]
		}

		pathParts := strings.Split(path, "/")
		if len(pathParts) < 3 {
			response.StatusCode = "404"
			return response
		}
		npm := pathParts[2]
		if npm != STUDENT_NPM {
			response.StatusCode = "404"
			return response
		}

		greeter := STUDENT_NAME
		if strings.HasPrefix(query, "name=") {
			nameVal := strings.TrimPrefix(query, "name=")
			if nameVal != "" {
				greeter = nameVal
			}
		}

		greet := GreetResponse{
			Student: Student{
				Nama: STUDENT_NAME,
				Npm:  STUDENT_NPM,
			},
			Greeter: greeter,
		}

		if req.Accept == "application/xml" {
			response.ContentType = "application/xml"
			response.Data = []byte(fmt.Sprintf(
				"<GreetResponse><Student><Nama>%s</Nama><Npm>%s</Npm></Student><Greeter>%s</Greeter></GreetResponse>",
				greet.Student.Nama, greet.Student.Npm, greet.Greeter,
			))
		} else {
			response.ContentType = "application/json"
			response.Data = []byte(fmt.Sprintf(
				`{"Student":{"Nama":"%s","Npm":"%s"},"Greeter":"%s"}`,
				greet.Student.Nama, greet.Student.Npm, greet.Greeter,
			))
		}

		response.StatusCode = "200"
		response = applyEncoding(req, response)
		return response
	}
	response.StatusCode = "404"
	return response
}


func RequestDecoder(bytestream []byte) HttpRequest {
    requestString := string(bytestream)
    lines := strings.Split(requestString, CRLF)

    var req HttpRequest

    if len(lines) > 0 {
        parts := strings.Split(lines[0], " ")
        if len(parts) == 3 {
            req.Method = parts[0]
            req.Uri = parts[1]
            req.Version = parts[2]
        }
    }

    req.AcceptEncoding = "none"

    for _, line := range lines[1:] {
        if line == "" {
            break
        }
        if strings.HasPrefix(line, "Host:") {
            req.Host = strings.TrimSpace(strings.TrimPrefix(line, "Host:"))
        } else if strings.HasPrefix(line, "Accept:") {
            req.Accept = strings.TrimSpace(strings.TrimPrefix(line, "Accept:"))
        } else if strings.HasPrefix(line, "Accept-Encoding:") {
            req.AcceptEncoding = strings.TrimSpace(strings.TrimPrefix(line, "Accept-Encoding:"))
        }
    }

    return req
}

func ResponseEncoder(res HttpResponse) []byte {
    statusLine := res.Version + " " + res.StatusCode + CRLF

    headers := ""
    if res.ContentType != "" {
        headers += "Content-Type: " + res.ContentType + CRLF
    }
    if res.ContentEncoding != "" && res.ContentEncoding != "none" {
        headers += "Content-Encoding: " + res.ContentEncoding + CRLF
    }
    if res.ContentLength > 0 {
        headers += "Content-Length: " + fmt.Sprint(res.ContentLength) + CRLF
    }

    response := statusLine + headers + CRLF

    return append([]byte(response), res.Data...)
}

func applyEncoding(req HttpRequest, res HttpResponse) HttpResponse {
	switch req.AcceptEncoding {
	case "gzip":
		var b bytes.Buffer
		gz := gzip.NewWriter(&b)
		_, _ = gz.Write(res.Data)
		gz.Close()
		res.Data = b.Bytes()
		res.ContentEncoding = "gzip"

	case "deflate":
		var b bytes.Buffer
		fl, _ := flate.NewWriter(&b, 6) // level 6 sesuai soal
		_, _ = fl.Write(res.Data)
		fl.Close()
		res.Data = b.Bytes()
		res.ContentEncoding = "deflate"

	case "none":
		res.ContentEncoding = ""

	default:
		var b bytes.Buffer
		gz := gzip.NewWriter(&b)
		_, _ = gz.Write(res.Data)
		gz.Close()
		res.Data = b.Bytes()
		res.ContentEncoding = "gzip"
	}

	res.ContentLength = len(res.Data)
	return res
}
