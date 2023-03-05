package apis

import (
	"database/sql"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"main/db"
	"main/token_controller"
	"main/utils"

	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
)

type PluginInfo struct {
	Name          string
	Author        string
	Description   string
	DownloadCount int
}

func GetPluginsList(c *gin.Context) {
	pluginName := c.Query("plugin_name")
	authorName := c.Query("author_name")

	var plugins []PluginInfo
	var rows *sql.Rows
	var err error

	if pluginName != "" && authorName != "" {
		// 根据插件名和作者名进行查询，其中插件名为模糊查询，作者名为精确查询
		s := "%"
		for _, v := range pluginName {
			s = s + string(v) + "%"
		}
		rows, err = db.Mysql.Query(`select name, author_name, description from view_plugins_info where author_name = ? and name like ?;`, authorName, s)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}
	} else {
		if pluginName != "" {
			// 根据提供的插件名进行模糊查询
			s := "%"
			for _, v := range pluginName {
				s = s + string(v) + "%"
			}
			rows, err = db.Mysql.Query(`select name, author_name, description from view_plugins_info where name like ?;`, s)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				c.String(http.StatusInternalServerError, "sever error.")
				return
			}
		} else if authorName != "" {
			// 根据提供的插件作者名进行精确查询
			rows, err = db.Mysql.Query(`select name, author_name, description from view_plugins_info where author_name = ?;`, authorName)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				c.String(http.StatusInternalServerError, "sever error.")
				return
			}
		} else {
			// 获取所有插件信息
		}
	}

	// 读取查询得到的数据，并返回
	for rows.Next() {
		var pluginInfo PluginInfo
		err := rows.Scan(&pluginInfo.Name, &pluginInfo.Author, &pluginInfo.Description)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}

		i, err := utils.GetPluginDownloadCount(pluginInfo.Author, pluginInfo.Name)
		if err != nil {
			if err == redis.ErrNil {
				pluginInfo.DownloadCount = 0
			} else {
				fmt.Printf("err: %v\n", err)
				c.String(http.StatusInternalServerError, "sever error.")
				return
			}
		} else {
			pluginInfo.DownloadCount = i
		}
		plugins = append(plugins, pluginInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"plugins": plugins,
	})
}

func GetPluginInfo(c *gin.Context) {
	username := c.Param("username")
	pluginName := c.Param("pluginName")

	pluginInfo := PluginInfo{Name: pluginName, Author: username}
	err := db.Mysql.QueryRow(`select description from view_plugins_info where author_name = ? and name = ?;`, username, pluginName).Scan(&pluginInfo.Description)
	if err != nil {
		if err == sql.ErrNoRows {
			c.String(http.StatusNotFound, "Not such plugin.")
			return
		}
		fmt.Printf("err: %v\n", err)
		c.String(http.StatusInternalServerError, "sever error.")
		return
	}

	i, err := utils.GetPluginDownloadCount(pluginInfo.Author, pluginInfo.Name)
	if err != nil {
		if err == redis.ErrNil {
			pluginInfo.DownloadCount = 0
		} else {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}
	} else {
		pluginInfo.DownloadCount = i
	}

	c.JSON(http.StatusOK, pluginInfo)
}

func DownloadPlugin(c *gin.Context) {
	username := c.Param("username")
	pluginName := c.Param("pluginName")

	var s string
	err := db.Mysql.QueryRow(`select 1 from view_plugins_info where name = ? and author_name = ? limit 1;`, pluginName, username).Scan(&s)
	if err != nil {
		if err == sql.ErrNoRows {
			c.String(http.StatusNotFound, "Not such plugin.")
			return
		}
		fmt.Printf("err: %v\n", err)
		c.String(http.StatusInternalServerError, "sever error.")
		return
	}

	f, err := os.Stat("data/plugins/" + utils.Md5(username+"salt") + "/" + utils.Md5(pluginName+"salt"))
	if f == nil || err != nil {
		c.String(http.StatusNotFound, "Not such plugin.")
		return
	}
	_, err = db.Redis.Do("incr", "downloadCount:"+username+":"+pluginName)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		c.String(http.StatusInternalServerError, "sever error.")
		return
	}
	c.Header("content-disposition", "attachment;filename="+pluginName+".dll")
	c.File("data/plugins/" + utils.Md5(username+"salt") + "/" + utils.Md5(pluginName+"salt"))
}

func AddPlugin(c *gin.Context) {
	token := c.GetHeader("Token")
	username := c.Param("username")
	description := c.PostForm("description")

	tokenInfo, err := token_controller.ParseToken(token, username)
	if err != nil {
		if err == token_controller.ErrIllegal || err == token_controller.ErrExpired || err == token_controller.ErrMismatch {
			c.String(http.StatusUnauthorized, err.Error())
		} else {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
		}
		return
	}

	f, err := c.FormFile("plugin")
	if err != nil {
		fmt.Printf("err: %v\n", err)
		c.String(http.StatusInternalServerError, "sever error.")
		return
	}
	pluginName := strings.Split(f.Filename, ".dll")[0]

	_, err = os.Stat("data/plugins/" + utils.Md5(username+"salt"))
	if err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir("data/plugins/"+utils.Md5(username+"salt"), 0777)
			if err != nil {
				c.String(http.StatusInternalServerError, "sever error.")
				return
			}
		} else {
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}
	}

	err = c.SaveUploadedFile(f, "data/plugins/"+utils.Md5(username+"salt")+"/"+utils.Md5(pluginName+"salt"))
	if err != nil {
		fmt.Printf("err: %v\n", err)
		c.String(http.StatusInternalServerError, "sever error.")
		return
	}
	_, err = db.Mysql.Exec(`insert into table_plugins(name, author_id, description) values (?, ?, ?);`, pluginName, tokenInfo.Uid, description)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		c.String(http.StatusInternalServerError, "sever error.")
		return
	}

	pluginInfo := PluginInfo{Name: pluginName, Author: username, Description: description, DownloadCount: 0}
	c.JSON(http.StatusOK, pluginInfo)
}

func UpdatePlugin(c *gin.Context) {
	token := c.GetHeader("Token")
	username := c.Param("username")
	pluginName := c.Param("pluginName")

	newPluginName := c.PostForm("name")
	description := c.PostForm("description")

	tokenInfo, err := token_controller.ParseToken(token, username)
	if err != nil {
		if err == token_controller.ErrIllegal || err == token_controller.ErrExpired || err == token_controller.ErrMismatch {
			c.String(http.StatusUnauthorized, err.Error())
		} else {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
		}
		return
	}

	// 判断插件是否存在于数据库之中
	var s string
	err = db.Mysql.QueryRow(`select 1 from view_plugins_info where name = ? and author_name = ? limit 1;`, pluginName, username).Scan(&s)
	if err != nil {
		if err == sql.ErrNoRows {
			c.String(http.StatusNotFound, "Not such plugin.")
			return
		}
		fmt.Printf("err: %v\n", err)
		c.String(http.StatusInternalServerError, "sever error.")
		return
	}

	// 判断是否有文件上传
	var f *multipart.FileHeader
	f, err = c.FormFile("plugin")
	if err == nil {
		newPluginName = strings.Split(f.Filename, ".dll")[0]
		err = c.SaveUploadedFile(f, "data/plugins/"+utils.Md5(username+"salt")+"/"+utils.Md5(newPluginName+"salt"))
		if err != nil {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}
	} else {
		if err != http.ErrMissingFile && err != http.ErrNotMultipart {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}
	}

	tx, err := db.Mysql.Begin()
	if err != nil {
		fmt.Printf("err: %v\n", err)
		c.String(http.StatusInternalServerError, "sever error.")
		return
	}

	if description != "" {
		_, err = tx.Exec(`update table_plugins set description = ? where name = ? and author_id = ?;`, description, pluginName, tokenInfo.Uid)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}
	}

	if newPluginName != "" && newPluginName != pluginName {
		// 判断该用户是否已有同名的插件
		err = db.Mysql.QueryRow(`select 1 from table_plugins where name = ? and author_id = ? limit 1;`, newPluginName, tokenInfo.Uid).Scan(&s)
		if err == nil {
			c.String(http.StatusBadRequest, "plugin name exists.")
			return
		} else if err != sql.ErrNoRows {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}

		if f != nil {
			// 新插件已下载，只需要移除旧插件
			err = os.Remove("data/plugins/" + utils.Md5(username+"salt") + "/" + utils.Md5(pluginName+"salt"))
			if err != nil {
				fmt.Printf("err: %v\n", err)
				c.String(http.StatusInternalServerError, "sever error.")
				return
			}
		} else {
			// 重命名旧插件
			err := os.Rename("data/plugins/"+utils.Md5(username+"salt")+"/"+utils.Md5(pluginName+"salt"), "data/plugins/"+utils.Md5(username+"salt")+"/"+utils.Md5(newPluginName+"salt"))
			if err != nil {
				fmt.Printf("err: %v\n", err)
				c.String(http.StatusInternalServerError, "sever error.")
				return
			}
		}

		// 判断插件是否已有下载次数
		_, err = redis.String(db.Redis.Do("get", "downloadCount:"+username+":"+pluginName))
		if err != nil {
			if err != redis.ErrNil {
				fmt.Printf("err: %v\n", err)
				c.String(http.StatusInternalServerError, "sever error.")
			}
		} else {
			_, err = db.Redis.Do("rename", "downloadCount:"+username+":"+pluginName, "downloadCount:"+username+":"+newPluginName)
			if err != nil && err != redis.ErrPoolExhausted {
				fmt.Printf("err: %v\n", err)
				c.String(http.StatusInternalServerError, "sever error.")
				return
			}
		}

		_, err = tx.Exec(`update table_plugins set name = ? where name = ? and author_id = ?;`, newPluginName, pluginName, tokenInfo.Uid)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		fmt.Printf("err: %v\n", err)
		c.String(http.StatusInternalServerError, "sever error.")
	} else {
		pluginInfo := PluginInfo{Author: username}
		if newPluginName != "" && newPluginName != pluginName {
			pluginInfo.Name = newPluginName
		} else {
			pluginInfo.Name = pluginName
		}
		err = db.Mysql.QueryRow(`select description from table_plugins where author_id = ? and name = ?;`, tokenInfo.Uid, pluginInfo.Name).Scan(&pluginInfo.Description)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}

		i, err := utils.GetPluginDownloadCount(pluginInfo.Author, pluginInfo.Name)
		if err != nil {
			if err == redis.ErrNil {
				pluginInfo.DownloadCount = 0
			} else {
				fmt.Printf("err: %v\n", err)
				c.String(http.StatusInternalServerError, "sever error.")
				return
			}
		} else {
			pluginInfo.DownloadCount = i
		}
		c.JSON(http.StatusOK, pluginInfo)
	}
}

func DeletePlugin(c *gin.Context) {
	token := c.GetHeader("Token")
	username := c.Param("username")
	pluginName := c.Param("pluginName")

	tokenInfo, err := token_controller.ParseToken(token, username)
	if err != nil {
		if err == token_controller.ErrIllegal || err == token_controller.ErrExpired || err == token_controller.ErrMismatch {
			c.String(http.StatusUnauthorized, err.Error())
		} else {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
		}
		return
	}

	// err = os.Remove("data/plugins/" + username + "/" + pluginName + ".dll")
	// if err != nil {
	// 	fmt.Printf("err: %v\n", err)
	// 	c.String(http.StatusInternalServerError, "sever error.")
	// 	return
	// }

	_, err = db.Mysql.Exec(`update table_plugins set delete_at = CURRENT_TIMESTAMP where name = ? and author_id = ?;`, pluginName, tokenInfo.Uid)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		c.String(http.StatusInternalServerError, "sever error.")
		return
	}

	c.Status(http.StatusNoContent)
}
