package server

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/nisainan/wstunnel/proxy"
	"github.com/nisainan/wstunnel/util"
	"github.com/urfave/cli"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

type Server struct {
	config      *Config
	upstream    *url.URL
	bufferPool  sync.Pool
	dialContext func(ctx context.Context, network, address string) (net.Conn, error)
}

func NewServer(c *cli.Context) *Server {
	server := &Server{
		config:     NewConfig(c),
		bufferPool: sync.Pool{New: func() interface{} { return make([]byte, 0, 32*1024) }},
	}
	upstream, err := url.Parse(server.config.Upstream)
	//upstream, err := url.Parse("http://127.0.0.1:80")
	if err != nil {
		log.Fatalf("upstream is wrong : %s\n", err)
	}
	server.upstream = upstream
	dialer := &net.Dialer{
		KeepAlive: 30 * time.Second,
	}
	server.dialContext = dialer.DialContext
	return server
}

func Run(c *cli.Context) {
	server := NewServer(c)
	srv := &http.Server{
		//Addr: "0.0.0.0:8888",
		Addr:    server.config.Addr,
		Handler: server,
	}
	go func() {
		if err := srv.ListenAndServeTLS(server.config.Cert, server.config.Key); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	fmt.Println("start", server.config.Addr)
	// 监听信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill)
	<-quit
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	targetAddr, clientHello, req, err := s.readWSRequest(r)
	if err != nil {
		log.Println("failed to readWSRequest: ", err)
		s.proxyCheatAddress(w, r)
		return
	}
	//fmt.Println(r.RemoteAddr, "==================")
	serverConn, err := net.Dial("tcp", targetAddr)
	//log.Println("connecting to ", targetAddr)
	if err != nil {
		log.Println("failed to connect to ", targetAddr, "error:", err)
		return
	}
	resp, err := s.generateInitialWSResponse(req)
	if err != nil {
		log.Println("error generating ws response", err)
		return
	}
	e, ok := w.(http.Hijacker)
	if !ok {
		log.Println("failed to hijack responseWriter")
		return
	}
	clientConn, _, err := e.Hijack()
	if err != nil {
		log.Printf("failed to hijack : %s\n", err)
		return
	}
	defer clientConn.Close()
	_, err = clientConn.Write(resp)
	if err != nil {
		log.Println("error writing ws response", err)
		return
	}
	_, err = serverConn.Write(clientHello)
	if err != nil {
		log.Println("error sending initial client request to the target:", err)
		return
	}
	proxy.TransparentProxy(clientConn, serverConn)
}

func (s *Server) readWSRequest(r *http.Request) (string, []byte, *http.Request, error) {
	log.SetPrefix(fmt.Sprintf("[%s] ", r.RemoteAddr))
	// 判断secretLink是否一致
	if r.URL.Path == s.config.SecretLink {
		if strings.ToLower(r.Header.Get("Connection")) != "upgrade" {
			return "", nil, nil, fmt.Errorf("Connection header expected: upgrade, got: %s\n",
				strings.ToLower(r.Header.Get("Connection")))
		}
		if strings.ToLower(r.Header.Get("Upgrade")) != "websocket" {
			return "", nil, nil, fmt.Errorf("Upgrade header expected: websocket, got: %s\n",
				strings.ToLower(r.Header.Get("Upgrade")))
		}
		clientHello, err := base64.StdEncoding.DecodeString(r.Header.Get("X-ReframerCH"))
		if err != nil {
			return "", nil, nil, err
		}
		targetAddr := r.Header.Get("X-TargetAddr")
		return targetAddr, clientHello, r, nil
	}
	return "", nil, nil, errors.New("SecretLink is wrong")
}

func (s *Server) proxyCheatAddress(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	upsConn, err := s.dialContext(ctx, "tcp", s.upstream.Host)
	if err != nil {
		log.Printf("failed to dial upstream : %s\n", err)
		return
	}
	//s.forwardResponseStream(upsConn, w, r)
	s.forwardResponse(upsConn, w, r)
}

// Removes hop-by-hop headers, and writes response into ResponseWriter.
func (s *Server) forwardResponse(conn net.Conn, w http.ResponseWriter, request *http.Request) {
	err := request.Write(conn)
	if err != nil {
		log.Printf("failed to write http request : %s\n", err)
		return
		//return http.StatusBadGateway, errors.New("failed to write http request: " + err.Error())
	}
	response, err := http.ReadResponse(bufio.NewReader(conn), request)
	if err != nil {
		log.Printf("failed to read http response : %s\n", err)
		return
		//return http.StatusBadGateway, errors.New("failed to read http response: " + err.Error())
	}
	request.Body.Close()
	if response != nil {
		defer response.Body.Close()
	}

	for header, values := range response.Header {
		for _, val := range values {
			w.Header().Add(header, val)
		}
	}
	util.RemoveHopByHop(w.Header())
	w.WriteHeader(response.StatusCode)
	buf := s.bufferPool.Get().([]byte)
	buf = buf[0:cap(buf)]
	io.CopyBuffer(w, response.Body, buf)
	return
}

func (s *Server) forwardResponseStream(conn net.Conn, w http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	wFlusher, ok := w.(http.Flusher)
	if !ok {
		log.Printf("failed to flush")
		return
		//return http.StatusInternalServerError, errors.New("ResponseWriter doesn't implement Flusher()")
	}
	w.WriteHeader(http.StatusOK)
	wFlusher.Flush()
	s.dualStream(conn, request.Body, w)
	return
}

// Copies data target->clientReader and clientWriter->target, and flushes as needed
// Returns when clientWriter-> target stream is done.
// Caddy should finish writing target -> clientReader.
func (s *Server) dualStream(target net.Conn, clientReader io.ReadCloser, clientWriter io.Writer) error {
	stream := func(w io.Writer, r io.Reader) error {
		// copy bytes from r to w
		buf := s.bufferPool.Get().([]byte)
		buf = buf[0:cap(buf)]
		_, _err := util.FlushingIoCopy(w, r, buf)
		if closeWriter, ok := w.(interface {
			CloseWrite() error
		}); ok {
			closeWriter.CloseWrite()
		}
		return _err
	}

	go stream(target, clientReader)
	return stream(clientWriter, target)
}

func (s *Server) generateInitialWSResponse(req *http.Request) ([]byte, error) {
	resp := http.Response{
		Status:           "101 Switching Protocols",
		StatusCode:       101,
		Proto:            "HTTP/1.1",
		ProtoMajor:       1,
		ProtoMinor:       1,
		Header:           http.Header{},
		Body:             nil,
		ContentLength:    0,
		TransferEncoding: nil,
		Close:            false,
		Uncompressed:     false,
		Trailer:          nil,
		Request:          nil,
		TLS:              nil,
	}
	resp.Header.Set("Upgrade", req.Header.Get("Upgrade"))
	resp.Header.Set("Connection", req.Header.Get("Connection"))
	//if len(hello) > 0 {
	//	resp.Header.Set("X-ReframerSH", base64.StdEncoding.EncodeToString(hello))
	//}
	//log.Println("GenerateInitialWSResponse")
	return httputil.DumpResponse(&resp, true)
}
