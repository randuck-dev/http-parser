package http

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/textproto"
	"sync"
)

const (
	HTTP11 = "HTTP/1.1"
)

var ErrUnsupportedHTTPVersion = errors.New("unsupported HTTP version")
var ErrIncompleteStatusLine = errors.New("incomplete StatusLine. Needs 3 parts")
var ErrStatusCodeOutsideOfRange = errors.New("statuscode is outside of allowed range 100-599")

var ErrInvalidHeaderFormat = errors.New("invalid header format detected. Expected format: \"key: value\"")

type Client interface {
	Get(string) (Response, error)
}

type Response struct {
	StatusLine StatusLine
	Headers    map[string]string
}

type StatusLine struct {
	HttpVersion  string
	StatusCode   uint16
	ReasonPhrase string
}

type HttpClient struct {
	net.Conn
}

func (hc *HttpClient) Get(uri string) (Response, error) {
	written, err := hc.Write([]byte(fmt.Sprintf("GET %s HTTP/1.1\nHost: localhost \r\n\r\n", uri)))
	if err != nil {
		slog.Error("Error while writing to connection", "err", err)
		return Response{}, err
	}

	slog.Debug("Written bytes", "written", written)
	return Response{}, nil
}

func Raw_http_parsing_docker_socket(docker_socket string, wg *sync.WaitGroup) {

	socket, err := dial(docker_socket)

	if err != nil {
		slog.Error("Unable to connect to socket", "err", err)
		return
	}

	client := HttpClient{socket}

	defer client.Close()

	wg.Wait()
}

func parseResponse(conn io.Reader) (Response, error) {
	reader := bufio.NewReader(conn)
	tp := textproto.NewReader(reader)

	var status_line StatusLine
	headers := make(map[string]string)
	line, err := tp.ReadLine()
	if err != nil {
		slog.Error("Error occurred while reading line", "err", err)
		return Response{}, err
	}
	status_line, err = parseStatusLine(line)
	if err != nil {
		slog.Error("Error when parsing status line", "err", err, "line", line)
		return Response{}, err
	}

	parsing_headers := true
	for {
		line, err := tp.ReadLine()
		if err != nil {
			return Response{}, err
		}

		if line == "" && parsing_headers {
			parsing_headers = false
			slog.Info("Finished parsing headers", "headers", headers)
			break
		} else if line != "" && parsing_headers {
			slog.Debug("Header", "header", line)

			key, value, err := parseHeader(line)
			if err != nil {
				slog.Error("Error when parsing header", "err", err, "line", line)
			}
			headers[key] = value
		}
	}

	return Response{
		StatusLine: status_line,
		Headers:    headers,
	}, nil
}

func dial(addr string) (net.Conn, error) {

	conn, err := net.Dial("unix", addr)

	if err != nil {
		return nil, err
	}

	return conn, nil
}
