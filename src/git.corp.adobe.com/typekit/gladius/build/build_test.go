package build

import (
	"fmt"
	"testing"
	"time"
	
	"github.com/garyburd/redigo/redis"
)

func redisCleanup() {
	conn, _ := redis.DialTimeout("tcp", ":6379", 0, 1*time.Second, 1*time.Second)
	
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
	
	defer redisCleanup()
}

func TestAll(t *testing.T) {
	buildA, _ := NewBuild("SHA_A")
	buildB, _ := NewBuild("SHA_B")
	
	conn, err := redis.DialTimeout("tcp", ":6379", 0, 1*time.Second, 1*time.Second)
	
	if err != nil {
		t.Errorf("unexpected error occurred %#v", err)
	}
	
	bm := NewBuildManager(conn)
	
	is_saved := bm.Save(buildA)
	
	if is_saved == false {
		t.Errorf("The build was not saved %#v", err)
	}
	
	is_saved = bm.Save(buildB)
	
	if is_saved == false {
		t.Errorf("The build was not saved %#v", err)
	}
	
	reply, err := redis.Values(conn.Do("SMEMBERS", "pending-builds"))
	
	if fmt.Sprintf("%s", reply[0]) != "build-SHA_A" {
		t.Errorf("Expected SHA_A, got %#s", fmt.Sprintf("%s", reply[0]))
	}
	
	if fmt.Sprintf("%s", reply[1]) != "build-SHA_B" {
		t.Errorf("Expected SHA_A, got %#s", fmt.Sprintf("%s", reply[0]))
	}
	
	if err != nil {
		t.Errorf("Could not retrieve the build %#v", err)
	}
	
	if reply == nil {
		t.Errorf("expected reply to be set, got %#v", reply)
	}

	defer conn.Close()
	
	defer redisCleanup()
}