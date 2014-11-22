package build

import (
	"errors"
	"fmt"
	"time"
	
	"github.com/garyburd/redigo/redis"
)

type Build struct {
	Sha string
}

type BuildManager struct {
  builds []*Build
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

func (m *BuildManager) All() []*Build {
	return m.builds
}

func (m *BuildManager) Save(build *Build) bool {
	conn, err := redis.DialTimeout("tcp", ":6379", 0, 1*time.Second, 1*time.Second)
	
	_, err = conn.Do("SADD", "pending-builds", fmt.Sprintf("build-%s", build.Sha))
	
	if err != nil {
		return false
	} else {
		return true
	}
}