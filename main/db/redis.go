package db

import (
	"fmt"

	"main/config"

	"github.com/garyburd/redigo/redis"
)

var Redis redis.Conn

func redisInit() {
	Redis, err = redis.Dial("tcp", config.Config["redis"]["host"]+":"+config.Config["redis"]["port"])
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
}

func flushPluginList() {
	_, err = Redis.Do("del", "pluginList")
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	rows, err := Mysql.Query(`select name from plugins_info;`)
	if err != nil {
		fmt.Println(err)
	}

	var s string
	for rows.Next() {
		rows.Scan(&s)
		Redis.Do("rpush", "pluginList", s)
	}
}
