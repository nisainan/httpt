package server

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
)

func ReadInitiaWSRequest(s *Server, clientConn net.Conn) (string, []byte, interface{}, error) {
	expectedH1Req := "GET /" + s.config.SecretLink
	log.SetPrefix(fmt.Sprintf("[%s] ", clientConn.RemoteAddr().String()))
	connReader := bufio.NewReader(clientConn)

	firstBytes, err := connReader.Peek(len(expectedH1Req))
	if err != nil {
		return "", nil, nil, err
	}

	if string(firstBytes) == expectedH1Req {
		req, err := http.ReadRequest(connReader)
		if err != nil {
			return "", nil, nil, err
		}

		if strings.ToLower(req.Header.Get("Connection")) != "upgrade" {
			return "", nil, nil, fmt.Errorf("Connection header expected: upgrade, got: %s\n",
				strings.ToLower(req.Header.Get("Connection")))
		}
		if strings.ToLower(req.Header.Get("Upgrade")) != "websocket" {
			return "", nil, nil, fmt.Errorf("Upgrade header expected: websocket, got: %s\n",
				strings.ToLower(req.Header.Get("Upgrade")))
		}

		clientHello, err := base64.StdEncoding.DecodeString(req.Header.Get("X-ReframerCH"))
		if err != nil {
			return "", nil, nil, err
		}
		targetAddr := req.Header.Get("X-TargetAddr")

		return targetAddr, clientHello, req, nil
	} else {
		// TODO: golang.org/x/net/http2 instead
		req, err := http.ReadRequest(connReader)
		if err != nil {
			log.Println(err)
			return "", nil, nil, err
		}
		reqBytes, err := httputil.DumpRequest(req, false)
		log.Println(string(reqBytes), err)
		return "", nil, nil, err
	}
}
