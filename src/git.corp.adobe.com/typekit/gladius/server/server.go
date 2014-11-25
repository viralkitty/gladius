// This package implements a simple HTTP server providing a REST API to a task handler.
//
// It provides four methods:
//
// 	GET    /task/          Retrieves all the tasks.
// 	POST   /task/          Creates a new task given a title.
// 	GET    /task/{taskID}  Retrieves the task with the given id.
// 	PUT    /task/{taskID}  Updates the task with the given id.
//
// Every method below gives more information about every API call, its parameters, and its results.
package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	
	"git.corp.adobe.com/typekit/gladius/build"
)

var conn, err = redis.DialTimeout("tcp", ":6379", 0, 1*time.Second, 1*time.Second)

var builds = build.NewBuildManager(conn)

const PathPrefix = "/build/"

func RegisterHandlers() {
	r := mux.NewRouter()
	r.HandleFunc(PathPrefix, errorHandler(ListBuilds)).Methods("GET")
	// r.HandleFunc(PathPrefix, errorHandler(NewTask)).Methods("POST")
	// r.HandleFunc(PathPrefix+"{id}", errorHandler(GetTask)).Methods("GET")
	http.Handle(PathPrefix, r)
}

// badRequest is handled by setting the status code in the reply to StatusBadRequest.
type badRequest struct{ error }

// notFound is handled by setting the status code in the reply to StatusNotFound.
type notFound struct{ error }

// errorHandler wraps a function returning an error by handling the error and returning a http.Handler.
// If the error is of the one of the types defined above, it is handled as described for every type.
// If the error is of another type, it is considered as an internal error and its message is logged.
func errorHandler(f func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := f(w, r)
		if err == nil {
			return
		}
		switch err.(type) {
		case badRequest:
			http.Error(w, err.Error(), http.StatusBadRequest)
		case notFound:
			http.Error(w, "task not found", http.StatusNotFound)
		default:
			log.Println(err)
			http.Error(w, "oops", http.StatusInternalServerError)
		}
	}
}

// ListTask handles GET requests on /task.
// There's no parameters and it returns an object with a Tasks field containing a list of tasks.
//
// Example:
//
//   req: GET /task/
//   res: 200 {"Tasks": [
//          {"ID": 1, "Title": "Learn Go", "Done": false},
//          {"ID": 2, "Title": "Buy bread", "Done": true}
//        ]}
func ListBuilds(w http.ResponseWriter, r *http.Request) error {
	res := struct{ Builds []*build.Build }{ builds.All() }
	return json.NewEncoder(w).Encode(res)
}

// NewTask handles POST requests on /task.
// The request body must contain a JSON object with a Title field.
// The status code of the response is used to indicate any error.
//
// Examples:
//
//   req: POST /task/ {"Title": ""}
//   res: 400 empty title
//
//   req: POST /task/ {"Title": "Buy bread"}
//   res: 200
// func NewTask(w http.ResponseWriter, r *http.Request) error {
// 	req := struct{ Title string }{}
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		return badRequest{err}
// 	}
// 	t, err := task.NewTask(req.Title)
// 	if err != nil {
// 		return badRequest{err}
// 	}
// 	return tasks.Save(t)
// }

// parseID obtains the id variable from the given request url,
// parses the obtained text and returns the result.
func parseID(r *http.Request) (int64, error) {
	txt, ok := mux.Vars(r)["id"]
	if !ok {
		return 0, fmt.Errorf("task id not found")
	}
	return strconv.ParseInt(txt, 10, 0)
}

// GetTask handles GET requsts to /task/{taskID}.
// There's no parameters and it returns a JSON encoded task.
//
// Examples:
//
//   req: GET /task/1
//   res: 200 {"ID": 1, "Title": "Buy bread", "Done": true}
//
//   req: GET /task/42
//   res: 404 task not found
// func GetTask(w http.ResponseWriter, r *http.Request) error {
// 	id, err := parseID(r)
// 	log.Println("Task is ", id)
// 	if err != nil {
// 		return badRequest{err}
// 	}
// 	t, ok := tasks.Find(id)
// 	log.Println("Found", ok)
//
// 	if !ok {
// 		return notFound{}
// 	}
// 	return json.NewEncoder(w).Encode(t)
// }