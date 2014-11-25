package build

import (
	"errors"
	"fmt"
	// "time"
	
	"github.com/garyburd/redigo/redis"
)

type Builds struct {
	builds []*Build
}

type Build struct {
	Sha string
}

type BuildManager struct {
  conn redis.Conn
}

func NewBuildManager(conn redis.Conn) *BuildManager {
	return &BuildManager{conn: conn}
}

func NewBuild(sha string) (*Build, error) {
	if sha == "" {
		return nil, errors.New("error occurred")
	} else {
		return &Build{Sha: sha}, nil		
	}
}

func BuildFromRawData(sha string) {
	
}

func (m *BuildManager) All() []*Build {	
	values, err := redis.Values(m.conn.Do("SMEMBERS", "pending-builds"))
	
	if err != nil {
		fmt.Printf("Error occurred: %#v", err)
	}
	
	builds := &Builds{}
	
	values, err = redis.Scan(values, builds)

	if err != nil {
		fmt.Printf("Error occurred: %#v", err)
	}
	
	fmt.Printf("TEST: %s", values[0])
	
	a, _ := NewBuild("xyz")
	b, _ := NewBuild("abc")

	return []*Build{a,b}
}

func (m *BuildManager) Save(build *Build) bool {
	_, err := m.conn.Do("SADD", "pending-builds", fmt.Sprintf("build-%s", build.Sha))
	
	if err != nil {
		return false
	} else {
		return true
	}
}