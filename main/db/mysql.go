package db

import (
	"database/sql"
	"fmt"

	"main/config"

	_ "github.com/go-sql-driver/mysql"
)

var (
	Mysql *sql.DB
	err   error
)

func mysqlInit() {
	Mysql, err = sql.Open("mysql", config.Config["mysql"]["username"]+":"+config.Config["mysql"]["password"]+"@tcp("+config.Config["mysql"]["host"]+":"+config.Config["mysql"]["port"]+")/plugin_market?charset=utf8")
	if err != nil {
		fmt.Println(err)
	}
}
