package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

type Routes struct {
	Scheduler *Scheduler
}

func (r *Routes) Builds(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")

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

		go b.Create(r.Scheduler)
	default:
	}
}
