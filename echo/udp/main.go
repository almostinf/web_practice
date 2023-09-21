package main

import (
	"net"
	"runtime"

	"github.com/alitto/pond"
	webpractice "github.com/almostinf/web_practice"
	server "github.com/almostinf/web_practice/echo/udp_server"
)

func main() {
	net.ResolveUDPAddr("udp", ":4000")
	serverConfig := server.Config{
		Port: ":4000",
		Transport: "udp",
	}

	workerPool := pond.New(runtime.NumCPU(), 100)
	server := server.New(serverConfig, workerPool, webpractice.GetLogger())

	server.Start()
	defer server.Stop()
}
