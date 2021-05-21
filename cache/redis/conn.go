package redis

import (
	"filestore-server/config"
	"fmt"
	"time"

	"github.com/garyburd/redigo/redis"
)

var pool *redis.Pool

func init() {
	pool = newRedisPool()
	fmt.Println("Init Redis Pool OK...")
}

// newRedisPool : 创建redis连接池
func newRedisPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle:     50,
		MaxActive:   30,
		IdleTimeout: 300 * time.Second,
		Dial: func() (redis.Conn, error) {
			// 1. 打开连接
			c, err := redis.Dial("tcp", config.RedisHost)
			if err != nil {
				fmt.Println(err)
				return nil, err
			}

			// 没设密码就无需验证
			// 2. 访问认证
			// if _, err = c.Do("AUTH", config.RedisPass); err != nil {
			// 	c.Close()
			// 	return nil, err
			// }
			return c, nil
		},
		TestOnBorrow: func(conn redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := conn.Do("PING")
			return err
		},
	}
}

func RedisPool() *redis.Pool {
	return pool
}
