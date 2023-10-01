package main

import (
	"log"
	"net/http"

	"github.com/almostinf/web_practice/fileserver/handler"
)

func main() {
	handler := handler.NewHandler()
	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	log.Println("Server is listening on port 8080")
	log.Fatal(server.ListenAndServe())
}
