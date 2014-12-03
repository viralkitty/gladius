package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"git.corp.adobe.com/typekit/gladius/server"
)

func main() {
	httpPort := os.Getenv("GLADIUS_HTTP_PORT")

	log.Printf("Listening on port %s", httpPort)

	http.ListenAndServe(fmt.Sprintf(":%s", httpPort), nil)
}
