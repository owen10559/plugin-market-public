package main

import (
	"main/apis"

	"github.com/gin-gonic/gin"
)

func main() {

	r := gin.Default()

	r.POST("/users", apis.AddUser)
	r.GET("/users/:username", apis.GetUserInfo)
	r.PATCH("/users/:username", apis.UpdateUserInfo)
	r.DELETE("/users/:username", apis.DeleteUser)

	r.GET("/plugins", apis.GetPluginsList)
	r.GET("/plugins/:username/:pluginName", apis.GetPluginInfo)
	r.GET("/plugins/:username/:pluginName/download", apis.DownloadPlugin)
	r.POST("/plugins/:username", apis.AddPlugin)
	r.PATCH("/plugins/:username/:pluginName", apis.UpdatePlugin)
	r.DELETE("/plugins/:username/:pluginName", apis.DeletePlugin)

	r.GET("/verification", apis.GetVerificationCode)
	r.POST("/token", apis.GetToken)

	r.Run(":10559")
}
