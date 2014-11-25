package build

import (
	"fmt"
	"testing"
	"time"
	
	"github.com/garyburd/redigo/redis"
)

var conn, _ = redis.DialTimeout("tcp", ":6379", 0, 1*time.Second, 1*time.Second)

func redisCleanup(conn redis.Conn) {	
	conn.Do("DEL", "pending-builds")
}


func TestNewBuild(t *testing.T) {
	
	sha := "xyz"
	
	build, _ := NewBuild(sha)
	
	if build.Sha != sha {
		t.Errorf("expected title %q, got %q", build, build.Sha)
	}
}

func TestNewBuildWithEmptySha(t *testing.T) {
		
	build, err := NewBuild("")
	
	if err == nil {
		t.Errorf("expected 'empty sha' error, got %#v", build)
	}
}


func TestSaveBuild(t *testing.T) {
	
	sha := "xyz"
	
	build, _ := NewBuild(sha)

	conn, err := redis.DialTimeout("tcp", ":6379", 0, 1*time.Second, 1*time.Second)
	
	if err != nil {
		t.Errorf("unexpected error occurred %#v", err)
	}
	
	bm := NewBuildManager(conn)
	
	is_saved := bm.Save(build)
	
	if is_saved == false {
		t.Errorf("The build was not saved %#v", err)
	}

	reply, err := redis.Values(conn.Do("SMEMBERS", "pending-builds"))
	
	if fmt.Sprintf("%s", reply[0]) != "build-xyz" {
		t.Errorf("Expected build-xyz, got %#v", reply)
	}
	
	if err != nil {
		t.Errorf("Could not retrieve the build %#v", err)
	}
	
	if reply == nil {
		t.Errorf("expected reply to be set, got %#v", reply)
	}

	defer conn.Close()
	
	defer redisCleanup(conn)\n\n
}

func TestAll(t *testing.T) {
	buildA, _ := NewBuild("SHA_A")
	buildB, _ := NewBuild("SHA_B")
	
	bm := NewBuildManager(conn)
	
	is_saved := bm.Save(buildA)	
	if is_saved == false {
		t.Errorf("The build was not saved %#v", buildA)
	}
	
	is_saved = bm.Save(buildB)
	if is_saved == false {
		t.Errorf("The build was not saved %#v", buildB)
	}
	
	allBuilds := bm.All()
	
	if len(allBuilds) != 2 {
		t.Errorf("error occurred, expected [buildA, buildB], got %#v", allBuilds)
	}
	
	if allBuilds[0].Sha != "build-SHA_A" {
		t.Errorf("expected buildA, got %#v", allBuilds[0])
	}
		
	defer redisCleanup(conn)
}