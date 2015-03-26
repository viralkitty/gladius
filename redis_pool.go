package main

import (
	"net"
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
	connection, err := redis.Dial(redisProtocol, net.JoinHostPort(redisIP.String(), redisPort))

	if err != nil {
		return nil, err
	}

	return connection, err
}

func redisTestOnBorrow(connection redis.Conn, t time.Time) error {
	_, err := connection.Do("PING")

	return err
}
