package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
)

var (
	addrFlag = flag.String("addr", "", "Redis address, including port")
)

func main() {
	flag.Parse()

	pool, err := redis.NewPool(redis.PoolConfig{Server: *addrFlag, UseTLS: false})
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	log.Println("pool created successfully")

	conn := pool.Get()
	defer conn.Close()

	ticker := time.NewTicker(1 * time.Second)
	quit := make(chan struct{})
	go func() {
		i := 0
		for {
			select {
			case <-ticker.C:
				conn.Do("SET", fmt.Sprintf("error:%d", i), 1, "EX", (10 * time.Minute).Seconds())
				log.Println("SET")
			case <-quit:
				ticker.Stop()
				return
			}
			i++
		}
	}()

	time.Sleep(10 * time.Minute)
	close(quit)
	log.Println("done")
}
