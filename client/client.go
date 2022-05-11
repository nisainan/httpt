package client

import (
	"bufio"
	"fmt"
	"github.com/nisainan/wstunnel/proxy"
	tls "github.com/refraction-networking/utls"
	"github.com/urfave/cli"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
)

type Client struct {
	tlsDialer *tls.Roller
	ln        net.Listener
	config    *Config
}

func NewClient(c *cli.Context) *Client {
	tlsDialer, err := tls.NewRoller()
	if err != nil {
		log.Fatalf("tls dialer is wrong : %s\n", err)
	}
	tlsDialer.HelloIDs = []tls.ClientHelloID{tls.HelloRandomizedNoALPN}
	client := &Client{
		tlsDialer: tlsDialer,
		config:    NewConfig(c),
	}
	return client
}

func (c *Client) ListenAndServe() error {
	ln, err := net.Listen("tcp", c.config.LocalAddr)
	if err != nil {
		return err
	}
	c.ln = ln
	fmt.Println("start", c.config.LocalAddr)
	return serve(c)
}

func serve(c *Client) error {
	for {
		clientConn, err := c.ln.Accept()
		if err != nil {
			log.Println("Failed to accept connection:", err)
			continue
		}
		go c.handleConn(clientConn)
	}
}

func (c *Client) handleConn(clientConn net.Conn) {
	defer clientConn.Close()

	log.SetPrefix("[" + clientConn.RemoteAddr().String() + "] ")

	req, err := http.ReadRequest(bufio.NewReader(clientConn))
	if err != nil {
		log.Println(err)
		return
	}

	if req.Method != http.MethodConnect {
		dump, dumpErr := httputil.DumpRequest(req, true)
		log.Println("unexpected request", dump, "\nerror:", dumpErr)
		return
	}

	res := &http.Response{StatusCode: http.StatusOK,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
	}
	err = res.Write(clientConn)
	if err != nil {
		log.Fatalln(err)
	}
	hello := make([]byte, 1024)
	n, err := clientConn.Read(hello)
	if err != nil {
		log.Println("failed to read client hello:", hello)
		return
	}
	//serverConn, _, err := dialWS(c.Config.Addr, map[string]string{"serverName": c.Config.Sni,
	//	"secretLink": c.Config.SecretLink}, req.RequestURI, hello[:n])
	serverConn, _, err := dialWS(c, req.RequestURI, hello[:n])
	if err != nil {
		log.Println(err)
		return
	}
	_ = proxy.TransparentProxy(clientConn, serverConn)
}

func Run(c *cli.Context) error {
	return NewClient(c).ListenAndServe()
}
