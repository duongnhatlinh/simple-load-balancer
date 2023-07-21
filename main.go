package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Server interface {
	Address() string
	IsAlive() bool
	Serve(rw http.ResponseWriter, req *http.Request)
}

type simpleServer struct {
	address string
	proxy   *httputil.ReverseProxy
}

func (s *simpleServer) Address() string {
	return s.address
}

func (s *simpleServer) IsAlive() bool {
	return true
}

func (s *simpleServer) Serve(rw http.ResponseWriter, req *http.Request) {
	s.proxy.ServeHTTP(rw, req)
}

func newSimpleServer(addr string) *simpleServer {
	serverUrl, err := url.Parse(addr)
	appErr(err)

	return &simpleServer{
		address: addr,
		proxy:   httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

func appErr(err error) {
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}

type loadBalancer struct {
	port            string
	servers         []Server
	roundRobinCount int
}

func newLoadBalancer(port string, s []Server) *loadBalancer {
	return &loadBalancer{
		port:            port,
		servers:         s,
		roundRobinCount: 0,
	}
}

func (lb *loadBalancer) getNextAvailableServer() Server {
	server := lb.servers[lb.roundRobinCount%len(lb.servers)]
	if !server.IsAlive() {
		lb.roundRobinCount++
		server = lb.servers[lb.roundRobinCount%len(lb.servers)]
	}
	lb.roundRobinCount++
	return server
}

func (lb *loadBalancer) serveProxy(rw http.ResponseWriter, req *http.Request) {
	server := lb.getNextAvailableServer()

	fmt.Printf("forward request to address %v\n'", server.Address())
	server.Serve(rw, req)
}

func main() {
	servers := []Server{
		newSimpleServer("https://www.youtube.com/"),
		newSimpleServer("https://www.duckduckgo.com"),
		newSimpleServer("https://www.google.com/"),
	}

	lb := newLoadBalancer("7000", servers)
	handleRedirect := func(rw http.ResponseWriter, req *http.Request) {
		lb.serveProxy(rw, req)
	}

	http.HandleFunc("/", handleRedirect)
	http.ListenAndServe(":"+lb.port, nil)

}
