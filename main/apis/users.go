package apis

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"main/db"
	"main/token_controller"
	"main/utils"

	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
)

type UserInfo struct {
	Name             string
	Email            string
	SelfIntroduction string
}

func GetVerificationCode(c *gin.Context) {
	email := c.Query("email")

	i, err := redis.Int(db.Redis.Do("ttl", "verificationCode:"+email))
	if err != nil {
		fmt.Printf("err: %v\n", err)
		c.Status(http.StatusInternalServerError)
	}
	// 10s内只允许发送一个验证码
	if 300-i < 10 {
		c.Status(http.StatusForbidden)
		return
	}

	err = utils.SendVerificationCode(email)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		c.Status(http.StatusInternalServerError)
	} else {
		c.Status(http.StatusAccepted)
	}
}

func AddUser(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	email := c.PostForm("email")
	code := c.PostForm("code")
	selfIntroduction := c.PostForm("selfIntroduction")

	// 判断邮箱验证码是否正确
	b, err := utils.VerifyCode(email, code)
	if !b {
		if err != nil {
			// 验证时系统错误
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		} else {
			// 验证码错误
			c.String(http.StatusUnauthorized, "verification code error.")
			return
		}
	}

	tx, err := db.Mysql.Begin()
	if err != nil {
		utils.ErrHandler(err, c)
		tx.Rollback()
		return
	}

	// 判断用户名是否已存在
	var s string
	err = tx.QueryRow(`select 1 from table_users where delete_at is null and name = ? limit 1;`, username).Scan(&s)
	if err == nil {
		c.String(http.StatusBadRequest, "username exists.")
		return
	} else if err != sql.ErrNoRows {
		fmt.Printf("err: %v\n", err)
		c.String(http.StatusInternalServerError, "sever error.")
		return
	}

	// 判断用户邮箱是否已存在
	err = tx.QueryRow(`select 1 from table_users where delete_at is null and email = ? limit 1;`, email).Scan(&s)
	if err == nil {
		c.String(http.StatusBadRequest, "email exists.")
		return
	} else if err != sql.ErrNoRows {
		fmt.Printf("err: %v\n", err)
		c.String(http.StatusInternalServerError, "sever error.")
		return
	}

	// 新增用户信息
	_, err = tx.Exec("insert into table_users (name, password, email, self_introduction) values (?, ?, ?, ?);", username, password, email, selfIntroduction)
	if err != nil {
		utils.ErrHandler(err, c)
		tx.Rollback()
		return
	}
	_, err = tx.Exec("unlock tables;")
	if err != nil {
		utils.ErrHandler(err, c)
		tx.Rollback()
		return
	}
	err = tx.Commit()
	if err != nil {
		utils.ErrHandler(err, c)
		tx.Rollback()
		return
	}

	var userInfo UserInfo
	err = db.Mysql.QueryRow(`select name, email, self_introduction from table_users where delete_at is null and name = ?;`, username).Scan(&userInfo.Name, &userInfo.Email, &userInfo.SelfIntroduction)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		c.String(http.StatusInternalServerError, "sever error.")
		return
	}
	// email信息隐藏
	s1 := userInfo.Email[:2]
	s2 := strings.Split(userInfo.Email, "@")[1]
	userInfo.Email = s1 + "***@" + s2

	c.JSON(http.StatusOK, userInfo)
}

func GetToken(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	email := c.PostForm("email")
	code := c.PostForm("code")
	exp := c.PostForm("exp")

	if username != "" {
		// 根据用户名密码获取token
		var truePassword string
		err := db.Mysql.QueryRow(`select password from table_users where delete_at is null and name = ?`, username).Scan(&truePassword)
		if err != nil {
			if err == sql.ErrNoRows {
				c.String(http.StatusUnauthorized, "username or password error.")
				return
			}
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}

		if password != truePassword {
			c.String(http.StatusUnauthorized, "username or password error.")
			return
		}
	} else {
		// 根据邮箱验证码获取token
		b, err := utils.VerifyCode(email, code)
		if b {
			err := db.Mysql.QueryRow(`select name from table_users where delete_at is null and email = ?`, email).Scan(&username)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				c.String(http.StatusInternalServerError, "sever error.")
				return
			}
		} else {
			if err != nil {
				// 验证时系统错误
				fmt.Printf("err: %v\n", err)
				c.String(http.StatusInternalServerError, "sever error.")
			} else {
				// 验证码错误
				c.String(http.StatusUnauthorized, "verification code error.")
			}
			return
		}
	}

	var token string
	var err error
	var uid string
	err = db.Mysql.QueryRow(`select id from table_users where name = ?;`, username).Scan(&uid)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		c.String(http.StatusInternalServerError, "sever error.")
	}
	if exp == "" {
		token, err = token_controller.CreateToken(uid, time.Hour*12)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}
	} else {
		i, err := strconv.ParseInt(exp, 10, 64)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}
		token, err = token_controller.CreateToken(uid, time.Duration(i)*time.Second)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}
	}
	c.String(http.StatusOK, token)
}

func GetUserInfo(c *gin.Context) {
	username := c.Param("username")
	token := c.GetHeader("Token")

	if token == "" {
		// 非用户本人查询
		userInfo := struct {
			Name             string
			SelfIntroduction string
		}{Name: username}
		err := db.Mysql.QueryRow(`select self_introduction from table_users where delete_at is null and name = ?;`, username).Scan(&userInfo.SelfIntroduction)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}
		c.JSON(http.StatusOK, userInfo)
	} else {
		_, err := token_controller.ParseToken(token, username)
		if err != nil {
			if err == token_controller.ErrIllegal || err == token_controller.ErrExpired || err == token_controller.ErrMismatch {
				fmt.Printf("err: %v\n", err)
				c.String(http.StatusUnauthorized, fmt.Sprintf("err: %v\n", err))
			} else {
				fmt.Printf("err: %v\n", err)
				c.String(http.StatusInternalServerError, "sever error.")
			}
			return
		}

		// 用户本人查询
		userInfo := struct {
			Name             string
			Email            string
			SelfIntroduction string
		}{Name: username}
		err = db.Mysql.QueryRow(`select email, self_introduction from table_users where delete_at is null and name = ?;`, username).Scan(&userInfo.Email, &userInfo.SelfIntroduction)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}
		// email信息隐藏
		s1 := userInfo.Email[:2]
		s2 := strings.Split(userInfo.Email, "@")[1]
		userInfo.Email = s1 + "***@" + s2

		c.JSON(http.StatusOK, userInfo)
	}
}

func UpdateUserInfo(c *gin.Context) {
	token := c.GetHeader("Token")
	username := c.Param("username")

	newUsername := c.PostForm("username")
	password := c.PostForm("password")
	newEmail := c.PostForm("email")
	newEmailcode := c.PostForm("newEmailCode")
	oldEmailcode := c.PostForm("oldEmailCode")
	selfIntroduction := c.PostForm("selfIntroduction")

	var s string
	var oldEmail string

	_, err := token_controller.ParseToken(token, username)
	if err != nil {
		if err == token_controller.ErrIllegal || err == token_controller.ErrExpired || err == token_controller.ErrMismatch {
			c.String(http.StatusUnauthorized, err.Error())
		} else {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
		}
		return
	}

	if newEmail != "" {
		// 验证新邮箱的验证码
		b, err := utils.VerifyCode(newEmail, newEmailcode)
		if !b {
			if err != nil {
				// 验证时系统错误
				fmt.Printf("err: %v\n", err)
				c.String(http.StatusInternalServerError, "sever error.")
			} else {
				// 验证码错误
				c.String(http.StatusUnauthorized, "new email verification code error.")
			}
			return
		}

		// 验证旧邮箱的验证码
		err = db.Mysql.QueryRow(`select email from table_users where delete_at is null and name = ?;`, username).Scan(&oldEmail)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}
		b, err = utils.VerifyCode(oldEmail, oldEmailcode)
		if !b {
			if err != nil {
				// 验证时系统错误
				fmt.Printf("err: %v\n", err)
				c.String(http.StatusInternalServerError, "sever error.")
			} else {
				// 验证码错误
				c.String(http.StatusUnauthorized, "old email verification code error.")
			}
			return
		}
	}

	tx, err := db.Mysql.Begin()
	if err != nil {
		fmt.Printf("err: %v\n", err)
		c.String(http.StatusInternalServerError, "sever error.")
		return
	}

	if newUsername != "" && newUsername != username {
		// 判断新的用户名是否已存在
		err = db.Mysql.QueryRow(`select 1 from table_users where delete_at is null and name = ? limit 1;`, newUsername).Scan(&s)
		if err == nil {
			c.String(http.StatusBadRequest, "username exists.")
			return
		} else if err != sql.ErrNoRows {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}

		_, err = tx.Exec("update table_users set name = ? where name = ?;", newUsername, username)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}
	}

	if password != "" {
		// 验证旧邮箱的验证码
		err = db.Mysql.QueryRow(`select email from table_users where delete_at is null and name = ?;`, username).Scan(&oldEmail)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}
		b, err := utils.VerifyCode(oldEmail, oldEmailcode)
		if !b {
			if err != nil {
				// 验证时系统错误
				fmt.Printf("err: %v\n", err)
				c.String(http.StatusInternalServerError, "sever error.")
			} else {
				// 验证码错误
				c.String(http.StatusUnauthorized, "verification code error.")
			}
			return
		}

		if newUsername == "" {
			_, err = tx.Exec("update table_users set password = ? where name = ?;", password, username)
		} else {
			_, err = tx.Exec("update table_users set password = ? where name = ?;", password, newUsername)
		}
		if err != nil {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}
	}

	if newEmail != "" && newEmail != oldEmail {
		// 判断新的用户邮箱是否已存在
		err = db.Mysql.QueryRow(`select 1 from table_users where delete_at is null and email = ? limit 1;`, newEmail).Scan(&s)
		if err == nil {
			c.String(http.StatusBadRequest, "email exists.")
			return
		} else if err != sql.ErrNoRows {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}

		if newUsername == "" {
			_, err = tx.Exec("update table_users set email = ? where name = ?;", newEmail, username)
		} else {
			_, err = tx.Exec("update table_users set email = ? where name = ?;", newEmail, newUsername)
		}
		if err != nil {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}
	}

	if selfIntroduction != "" {
		if newUsername == "" {
			_, err = tx.Exec("update table_users set self_introduction = ? where name = ?;", selfIntroduction, username)
		} else {
			_, err = tx.Exec("update table_users set self_introduction = ? where name = ?;", selfIntroduction, newUsername)
		}
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
		var userInfo struct {
			Name             string
			Email            string
			SelfIntroduction string
		}
		if newUsername == "" {
			err = db.Mysql.QueryRow(`select name, email, self_introduction from table_users where delete_at is null and name = ?;`, username).Scan(&userInfo.Name, &userInfo.Email, &userInfo.SelfIntroduction)
		} else {
			err = db.Mysql.QueryRow(`select name, email, self_introduction from table_users where delete_at is null and name = ?;`, newUsername).Scan(&userInfo.Name, &userInfo.Email, &userInfo.SelfIntroduction)
		}
		if err != nil {
			fmt.Printf("err: %v\n", err)
			c.String(http.StatusInternalServerError, "sever error.")
			return
		}
		// email信息隐藏
		s1 := userInfo.Email[:2]
		s2 := strings.Split(userInfo.Email, "@")[1]
		userInfo.Email = s1 + "***@" + s2

		c.JSON(http.StatusOK, userInfo)
	}
}

func DeleteUser(c *gin.Context) {
	token := c.GetHeader("Token")
	username := c.Param("username")

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

	err = os.RemoveAll("data/plugins/" + username)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		c.String(http.StatusInternalServerError, "sever error.")
		return
	}

	tx, err := db.Mysql.Begin()
	if err != nil {
		fmt.Printf("err: %v\n", err)
		c.String(http.StatusInternalServerError, "sever error.")
		return
	}
	_, err = tx.Exec(`update table_plugins set delete_at = CURRENT_TIMESTAMP where author_id = ?;`, tokenInfo.Uid)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		c.String(http.StatusInternalServerError, "sever error.")
		return
	}
	_, err = tx.Exec(`update table_users set delete_at = CURRENT_TIMESTAMP where delete_at is null and name = ?;`, username)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		c.String(http.StatusInternalServerError, "sever error.")
		return
	}

	err = tx.Commit()
	if err != nil {
		fmt.Printf("err: %v\n", err)
		c.String(http.StatusInternalServerError, "sever error.")
	} else {
		c.Status(http.StatusNoContent)
	}
}
