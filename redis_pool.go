package main

import (
	"net"
	"os"
	"time"

	redis "github.com/garyburd/redigo/redis"
)

func NewRedisPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle:      redisMaxIdle,
		IdleTimeout:  redisIdleTimeout * time.Second,
		Dial:         redisDial,
		TestOnBorrow: redisTestOnBorrow,
	}
}

func redisDial() (redis.Conn, error) {
	ip := os.Getenv("REDIS_PORT_6379_TCP_ADDR")
	port := os.Getenv("REDIS_PORT_6379_TCP_PORT")
	protocol := os.Getenv("REDIS_PORT_6379_TCP_PROTO")

	return redis.Dial(protocol, net.JoinHostPort(ip, port))
}

func redisTestOnBorrow(connection redis.Conn, t time.Time) error {
	_, err := connection.Do("PING")

	return err
}
