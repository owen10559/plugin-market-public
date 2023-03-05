package utils

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"main/config"
	"main/db"

	"github.com/garyburd/redigo/redis"
	"gopkg.in/gomail.v2"
)

var d *gomail.Dialer

func smtpInit() {
	port, err := strconv.Atoi(config.Config["smtp"]["port"])
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
	d = gomail.NewDialer(config.Config["smtp"]["host"], port, config.Config["smtp"]["account"], config.Config["smtp"]["password"])
}

func SendMail(receiver string, subject string, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", m.FormatAddress(config.Config["smtp"]["account"], config.Config["smtp"]["name"]))
	m.SetHeader("To", receiver)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)
	err := d.DialAndSend(m)
	return err
}

func SendVerificationCode(account string) error {
	rand.Seed(time.Now().Unix())
	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	err := SendMail(account, "验证码", code)
	if err != nil {
		return err
	}
	_, err = db.Redis.Do("set", "verificationCode:"+account, code, "ex", "300")
	return err
}

func VerifyCode(account string, code string) (bool, error) {
	trueCode, err := redis.String(db.Redis.Do("get", "verificationCode:"+account))
	if err != nil {
		if err == redis.ErrNil {
			return false, nil
		} else {
			return false, err
		}
	}
	if trueCode == code {
		db.Redis.Do("del", "verificationCode:"+account)
		return true, nil
	} else {
		return false, nil
	}
}
