package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

type Routes struct {
}

func NewRoutes() *Routes {
	return &Routes{}
}

func (r *Routes) Builds(w http.ResponseWriter, req *http.Request) {
	var (
		err  error
		body []byte
	)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	switch req.Method {
	case "OPTIONS":
		return
	case "GET":
		w.Header().Set("Content-Type", "application/json")

		log.Printf("GET /builds")

		builds := AllBuilds()
		body, err = json.Marshal(builds)

		if err != nil {
			log.Printf("Could not marshal the builds: %v", err)
			return
		}

		w.Write(body)
	case "POST":
		w.Header().Set("Content-Type", "application/json")

		log.Printf("POST /builds")

		b := NewBuild()
		body, err = ioutil.ReadAll(req.Body)

		if err != nil {
			log.Printf("Could not read the request body: %v", err)
			return
		}

		err = json.Unmarshal(body, b)

		if err != nil {
			log.Fatal("Could not unmarshal the request body: ", err)
		}

		body, err = json.Marshal(*b)

		go b.Build()

		w.Write(body)
	default:
	}
}
