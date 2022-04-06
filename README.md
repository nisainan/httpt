# httpt
Prototype of circumvention system described in https://www.usenix.org/conference/foci20/presentation/frolov



This proxy is based on websocket, so you can hide your server behind cdn which suport websocket protocol like Cloudflare.Also, when someone try to access your server in https protocol ,server will response a http2 website that you configured.Go browse all the things!



~~~shell
server:
	./httpt --type server --config server.yaml
	
client:
  ./httpt --type client --config client.yaml
~~~

