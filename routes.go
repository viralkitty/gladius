package main

import (
	"code.google.com/p/go-uuid/uuid"
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type Routes struct {
	Scheduler *Scheduler
}

func (r *Routes) Builds(w http.ResponseWriter, req *http.Request) {
	var body []byte

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")

	switch req.Method {
	case "GET":
		log.Printf("GET /builds")

		keys, err := redis.Strings(redisCli.Do("KEYS", "pugio:builds:typekit:*"))
		keysWithoutPrefix := make([]string, len(keys))

		if err != nil {
			log.Printf("Could not get keys: %s", err)
			return
		}

		for i, key := range keys {
			keysWithoutPrefix[i] = strings.Replace(key, "pugio:builds:typekit:", "", 1)
		}

		body, err = json.Marshal(keysWithoutPrefix)

		if err != nil {
			log.Printf("Could not marshal object")
			return
		}

		w.Write(body)
	case "POST":
		log.Printf("POST /builds")

		var b Build

		body, err := ioutil.ReadAll(req.Body)

		if err != nil {
			log.Fatal("Could not read the request body: ", err)
		}

		err = json.Unmarshal(body, &b)

		if err != nil {
			log.Fatal("Could not unmarshal the request body: ", err)
		}

		b.Id = uuid.New()

		body, err = json.Marshal(b)

		log.Printf("%+v", b)

		go b.Create(r.Scheduler)
		go redisCli.Do("HMSET", fmt.Sprintf("pugio:builds:typekit:%s", b.Id), "app", b.App, "branch", b.Branch)

		w.Write(body)
	default:
	}
}
