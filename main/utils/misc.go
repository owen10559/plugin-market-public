package utils

import (
	"crypto/md5"
	"fmt"
	"net/http"

	"main/db"

	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
)

func Md5(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

func GetPluginDownloadCount(username string, pluginName string) (int, error) {
	i, err := redis.Int(db.Redis.Do("get", "downloadCount:"+username+":"+pluginName))
	if err != nil {
		return 0, err
	} else {
		return i, nil
	}
}

func ErrHandler(err error, c *gin.Context) {
	// fmt.Printf("err: %v\n", errors.WithStack(err))
	c.Error(err)
	c.String(http.StatusInternalServerError, "sever error.")
}
