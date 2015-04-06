package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	//	"reflect"
	"strings"

	redis "github.com/garyburd/redigo/redis"
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
	log.Printf("%s %s", req.Method, req.URL.Path)

	conn := redisPool.Get()

	defer conn.Close()

	switch req.Method {
	case "OPTIONS":
		w.WriteHeader(http.StatusOK)

		return
	case "GET":
		var keyBuffer bytes.Buffer

		urlPath := strings.Split(req.URL.Path, "/")

		keyBuffer.WriteString("pugio:builds")

		switch len(urlPath) {
		case 2:
			builds := []*Build{}
			values, _ := redis.Values(conn.Do("LRANGE", keyBuffer.String(), -1, 10))

			for _, value := range values {
				var build Build

				bytes, err := redis.Bytes(value, err)

				if err != nil {
					log.Printf(err.Error())

					continue
				}

				err = json.Unmarshal(bytes, &build)

				if err != nil {
					log.Printf(err.Error())
				} else {
					builds = append(builds, &build)
				}
			}

			body, _ = json.Marshal(&builds)
		case 3:
			keyBuffer.WriteString(":")
			keyBuffer.WriteString(urlPath[2])

			reply, err := conn.Do("GET", keyBuffer.String())

			if err != nil {
				log.Printf("Could not get key: %s", keyBuffer.String())
				w.WriteHeader(http.StatusInternalServerError)

				return
			} else {
				var (
					build     Build
					logBuffer bytes.Buffer
				)

				bytes, err := redis.Bytes(reply, err)

				if err != nil {
					log.Printf(err.Error())
					w.WriteHeader(http.StatusInternalServerError)

					return
				}

				err = json.Unmarshal(bytes, &build)

				if err != nil {
					log.Printf(err.Error())
					w.WriteHeader(http.StatusInternalServerError)

					return
				}

				values, err := redis.Values(conn.Do("LRANGE", build.RedisLogKey(), 0, -1))

				if err != nil {
					log.Printf(err.Error())
					w.WriteHeader(http.StatusInternalServerError)

					return
				}

				for index, value := range values {
					bytes, _ := redis.Bytes(value, err)

					if index != 0 {
						logBuffer.WriteString("\n")
					}

					logBuffer.WriteString(string(bytes[:]))
				}

				build.Log = logBuffer.String()

				body, err = json.Marshal(&build)

				if err != nil {
					log.Printf(err.Error())
					w.WriteHeader(http.StatusInternalServerError)

					return
				}
			}
		default:
			break
		}

		w.WriteHeader(http.StatusOK)
	case "POST":
		body, err = ioutil.ReadAll(req.Body)

		if err != nil {
			log.Printf("Could not read the request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		b := NewBuild()
		err = json.Unmarshal(body, b)

		if err != nil {
			log.Printf("Could not unmarshal the request body: ", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		err = b.Save()

		if err != nil {
			log.Printf("Could not save the build: ", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		body, err = json.Marshal(*b)

		if err != nil {
			log.Printf("Could not marshal the build: ", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		_, err = conn.Do("LPUSH", "pugio:builds", body)

		if err != nil {
			log.Print(err)
		}

		go b.Build()

		w.WriteHeader(http.StatusCreated)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}
