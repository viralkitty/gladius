package main

import (
	"os"
	"log"
	"net/http"

	"git.corp.adobe.com/typekit/gladius/server"
)

func main() {
	httpPort := os.Getenv('GLADIUS_HTTP_PORT')

	log.Printf("Starting Gladius")
	server.RegisterHandlers()
	http.ListenAndServe(":8080", nil)
	log.Printf("Listening on port %s", httpPort)
}
