package main

import (
	"runtime"

	"github.com/alitto/pond"
	webpractice "github.com/almostinf/web_practice"
	server "github.com/almostinf/web_practice/echo/tcp_server"
)

func main() {
	serverConfig := server.Config{
		Transport: "tcp",
		URL:       "localhost:4000",
	}
	workerPool := pond.New(runtime.NumCPU(), 100)
	server := server.New(serverConfig, workerPool, webpractice.GetLogger())

	server.Start()
	defer server.Stop()
}
