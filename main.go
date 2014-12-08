package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	listenAt := fmt.Sprintf(":%s", os.Getenv("GLADIUS_HTTP_PORT"))

	log.Printf("Listening at %s", listenAt)

	http.HandleFunc("/builds", Builds)
	http.ListenAndServe(listenAt, nil)
}
