package main

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
)

const (
	SERVER_TYPE = "tcp"
	BUFFER_SIZE = 2048
	CRLF = "\r\n"
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
	var urlInput string
	var contentType string
	var encodingType string

	fmt.Print("Input URL: ")
	fmt.Scan(&urlInput)

	fmt.Print("Input Content Type: ")
	fmt.Scan(&contentType)

	fmt.Print("Input Accept Encoding (write \"none\" if no special encoding can be accepted): ")
	fmt.Scan(&encodingType)

	var request HttpRequest
	u, err := url.Parse(urlInput)
	if err != nil {
		panic(err)
	}
	request.Method = "GET"
	request.Uri = u.RequestURI()
	request.Version = "HTTP/1.1"
	request.Host = u.Host
	request.Accept = contentType
	request.AcceptEncoding = encodingType
	connection, err := net.Dial(SERVER_TYPE, u.Host)
    if err != nil {
        fmt.Printf("Error connecting to server: %v\n", err)
        return
    }
    defer connection.Close()
    response := Fetch(request, connection)
    fmt.Print("Status Code: " + response.StatusCode + CRLF)
    if response.ContentEncoding != "none" {
        fmt.Print("Encoded: " + response.ContentEncoding + CRLF)
    }
    fmt.Println("Body: " + string(response.Data))
    if response.ContentType != "text/html" {
        var parsedData GreetResponse
        err := json.Unmarshal(response.Data, &parsedData)
        if err != nil {
            panic(err)
        }
        fmt.Print(CRLF + "Parsed: ", parsedData)
    }
}

func Fetch(req HttpRequest, connection net.Conn) HttpResponse {
	requestEncoded := RequestEncoder(req)

    _, err := connection.Write(requestEncoded)
    if err != nil {
        fmt.Printf("Error sending request: %v\n", err)
        return HttpResponse{}
    }

    buffer := make([]byte, BUFFER_SIZE)
    n, err := connection.Read(buffer)
    if err != nil {
        fmt.Printf("Error reading response: %v\n", err)
        return HttpResponse{}
    }

    return ResponseDecoder(buffer[:n])
}

func ResponseDecoder(bytestream []byte) HttpResponse {
    responseDecoded := string(bytestream)
    lines := strings.Split(responseDecoded, CRLF)

    var res HttpResponse

    if len(lines) > 0 {
        parts := strings.SplitN(lines[0], " ", 3)
        if len(parts) >= 2 {
            res.Version = parts[0]
            res.StatusCode = parts[1]
            if len(parts) == 3 {
                res.StatusCode += " " + parts[2]
            }
        }
    }

    res.ContentEncoding = "none"

    bodyIndex := 0
    for i, line := range lines[1:] {
        if line == "" {
            bodyIndex = i + 2
            break
        }
        if strings.HasPrefix(line, "Content-Type:") {
            res.ContentType = strings.TrimSpace(strings.TrimPrefix(line, "Content-Type:"))
        } else if strings.HasPrefix(line, "Content-Encoding:") {
            res.ContentEncoding = strings.TrimSpace(strings.TrimPrefix(line, "Content-Encoding:"))
        } else if strings.HasPrefix(line, "Content-Length:") {
            lengthStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
            length, err := strconv.Atoi(lengthStr)
            if err == nil {
                res.ContentLength = length
            }
        }
    }

    if bodyIndex < len(lines) {
        body := strings.Join(lines[bodyIndex:], CRLF)
        res.Data = []byte(body)

        res.Data = decodeCompressedData(res.Data, res.ContentEncoding)
    }

    return res
}


func RequestEncoder(req HttpRequest) []byte {
	requestLine := req.Method + " " + req.Uri + " " + req.Version + CRLF
	headers := "Host: " + req.Host + CRLF + "Accept: " + req.Accept + CRLF
	if req.AcceptEncoding != "none" {
		headers += "Accept-Encoding: " + req.AcceptEncoding + CRLF
	}

	return []byte(requestLine + headers + CRLF)
}

func decodeCompressedData(data []byte, encoding string) []byte {
    switch encoding {
    case "gzip":
        return decodeGzip(data)
    case "deflate":
        return decodeDeflate(data)
    default:
        return data
    }
}

func decodeGzip(data []byte) []byte {
    reader, err := gzip.NewReader(bytes.NewReader(data))
    if err != nil {
        fmt.Printf("Error creating gzip reader: %v\n", err)
        return data
    }
    defer reader.Close()

    decompressed, err := io.ReadAll(reader)
    if err != nil {
        fmt.Printf("Error decompressing gzip data: %v\n", err)
        return data
    }

    return decompressed
}

func decodeDeflate(data []byte) []byte {
    reader := flate.NewReader(bytes.NewReader(data))
    defer reader.Close()

    decompressed, err := io.ReadAll(reader)
    if err != nil {
        fmt.Printf("Error decompressing deflate data: %v\n", err)
        return data
    }

    return decompressed
}
