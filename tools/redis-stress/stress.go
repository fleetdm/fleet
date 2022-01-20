package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
)

var (
	addrFlag       = flag.String("addr", "", "Redis address, including port")
	indexStartFlag = flag.Int("index-start", 1, "Index to start from when inserting keys")
	debugFlag      = flag.Bool("debug", false, "Print debug logs")
	waitFlag       = flag.Duration("wait", 10*time.Minute, "Amount of time to do SETs")
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
		if indexStartFlag != nil {
			i = *indexStartFlag
		}
		for {
			select {
			case <-ticker.C:
				_, err := conn.Do("SET", fmt.Sprintf("error:%d", i), 1, "EX", (10 * time.Minute).Seconds())
				if debugFlag != nil && *debugFlag {
					log.Println("SET", i)
					if err != nil {
						log.Println("err", err)
					}
				}
			case <-quit:
				ticker.Stop()
				return
			}
			i++
		}
	}()

	time.Sleep(*waitFlag)
	close(quit)
	log.Println("done")
}
