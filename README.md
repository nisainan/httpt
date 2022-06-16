![logo](logo.png)

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)  [![GoDoc](https://godoc.org/github.com/cloudflare/cfssl?status.svg)](https://pkg.go.dev/github.com/nisainan/wstunnel)

A proxy based on native ws protocal. But can response a http2 website that you configured without authorization to hide your proxy.

## Features

- Proxy based on websocket
- TLS support
- Secret link
- Camouflage traffic
- CDN
- ClientHello fingerprinting resistance(working on it)

## Installing

~~~shell
$ git clone https://github.com/nisainan/wstunnel.git
$ cd wstunnel
$ go build
~~~

Edit `client/config.yaml` and `server/config.yaml` with your own data

**client/config.yaml**

~~~yaml
local-addr: "127.0.0.1:9999" # http listen address in your local machine
server-addr: "domain:443" # remote server address
sni: "ws.sekiro.vip" # remote server sni
secret-link: "/secretLink" # websocket secret link
~~~

**server/config.yaml**

~~~yaml
cert: "xxxx" # cert file localtion
key: "xxxx" # key file localtion
addr: "0.0.0.0:443" # listen address
secret-link: "/secretLink" # websocket secret link,same as client's
upstream: "http://127.0.0.1:80" # cheat-host, make sure this server works
~~~

## Usage

**client**

~~~shell
./wstunnel --type client --config client.yaml
~~~

**server**

~~~shell
./wstunnel --type server --config server.yaml
~~~

1. Use SwitchyOmega in your browser
2. Add a http proxy whith your client address
3. Congratulations,Go browse all the things!

## CDN

This proxy is based on websocket, so you can hide your server behind cdn which suport websocket protocol like Cloudflare.Also, when someone try to access your server in https protocol ,server will response a http2 website that you configured

## License

WsTunnel source code is available under the MIT [License](https://github.com/nisainan/wstunnel/blob/master/LICENSE).

## Thanks

[httpt](https://github.com/sergeyfrolov/httpt)
