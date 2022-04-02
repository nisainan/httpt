package client

import (
	"bufio"
	"encoding/base64"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
)

func dialWS(c *Client, targetAddr string, hello []byte) (net.Conn, []byte, error) {
	//    // if reframerID is nil, creates new connection to reframer server
	//    establish TLS to server
	//    send HTTP/1.1 WS request. It will include the ID to reconnect, if reconnecting
	//    WS response will include in the headers the InitialState or ReconnectState
	conn, err := c.tlsDialer.Dial("tcp", c.config.ServerAddr, c.config.Sni)
	if err != nil {
		log.Println(err)
		return nil, nil, err
	}

	//log.Printf("[uTLS] mimicking %v. ALPN offered: %v, chosen: %v\n",
	//	conn.ClientHelloID.Str(), conn.HandshakeState.Hello.AlpnProtocols, conn.HandshakeState.ServerHello.AlpnProtocol)

	switch conn.HandshakeState.ServerHello.AlpnProtocol {
	case "http/1.1", "":
		req, err := http.NewRequest("GET", c.config.SecretLink, nil)
		if err != nil {
			log.Println(err)
			return nil, nil, err
		}
		req.Host = c.config.Sni

		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("X-ReframerCH", base64.StdEncoding.EncodeToString(hello))
		req.Header.Set("X-TargetAddr", targetAddr)

		// TODO: req.Header.Set("X-Padding", "[][]")

		if err := req.Write(conn); err != nil {
			log.Printf("failed to write WebSocket Upgrade Request: %v\n", err)
			return nil, nil, err
		}
		//log.Println("DEBUG wrote req, err", err)

		resp, err := http.ReadResponse(bufio.NewReader(conn), req)
		if err != nil {
			log.Println(err)
			return nil, nil, err
		}

		if resp.Status == "101 Switching Protocols" &&
			strings.ToLower(resp.Header.Get("Upgrade")) == "websocket" &&
			strings.ToLower(resp.Header.Get("Connection")) == "upgrade" {

			serverHello, err := base64.StdEncoding.DecodeString(resp.Header.Get("X-ReframerSH"))
			if err != nil {
				return nil, nil, err
			}

			return conn, serverHello, nil
		} else {
			//respBytes, err := httputil.DumpResponse(resp, false)
			//if err != nil {
			//	log.Println(err)
			//	return nil, nil, err
			//}
			err = errors.New("Got unexpected response: status:" + resp.Status)
			//log.Println(err)
			return nil, nil, err
		}
	case "h2":
		return nil, nil, errors.New("http2 is not implemented yet")
	default:
		return nil, nil, errors.New("Unknown ALPN: " + conn.HandshakeState.ServerHello.AlpnProtocol)
	}
}
