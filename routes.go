package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

func Builds(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "POST":
		var b Build

		body, err := ioutil.ReadAll(req.Body)

		if err != nil {
			log.Fatal("Could not read the request body: ", err)
		}

		err = json.Unmarshal(body, &b)

		if err != nil {
			log.Fatal("Could not unmarshal the request body: ", err)
		}

		go b.Create()
	default:
	}
}
