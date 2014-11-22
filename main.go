package main

import (
	"net/http"

	"git.corp.adobe.com/typekit/gladius/server"
)

func main() {
	server.RegisterHandlers()
	http.Handle("/", http.FileServer(http.Dir("static")))
	http.ListenAndServe(":8080", nil)