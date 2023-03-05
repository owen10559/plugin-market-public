package apis_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"main/token_controller"

	"github.com/garyburd/redigo/redis"
)

var (
	Redis redis.Conn
	err   error
)

func init() {
	Redis, err = redis.Dial("tcp", "redis:6379")
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
}

func showDetail(req *http.Request, res *http.Response, body ...[]byte) {
	fmt.Printf("req.Body: %v\n", req.Body)
	fmt.Printf("res.Status: %v\n", res.Status)
	var s string
	if len(body) == 1 {
		s = string(body[0])
	} else {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Printf("err: %v\n", err)
		}
		s = string(body)
	}
	fmt.Println("res.Body: " + s)
}

func GetVerificationCode(account string) (string, error) {
	url := "http://127.0.0.1:10559/verification?email=" + account
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		return "", err
	}
	_, err = client.Do(req)
	if err != nil {
		return "", err
	}

	code, err := redis.String(Redis.Do("get", "verificationCode:"+account))
	if err != nil {
		return "", err
	}
	return code, nil
}

func GetTokenByPassword(username string, password string) (string, error) {
	url := "http://127.0.0.1:10559/token"
	method := "POST"

	payload := strings.NewReader("username=" + username + "&password=" + password)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func TestGetVerificationCode(t *testing.T) {
	url := "http://127.0.0.1:10559/verification?email=owen10559@qq.com"
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Error(err)
		return
	}

	res, err := client.Do(req)
	if err != nil {
		t.Error(err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 202 {
		t.Error()
		showDetail(req, res)
	}
}

func TestAddUser(t *testing.T) {
	code, err := GetVerificationCode("user1@email.com")
	if err != nil {
		t.Error(err)
		return
	}

	url := "http://127.0.0.1:10559/users"
	method := "POST"

	payload := strings.NewReader("username=user1&password=12345&selfIntroduction=自我介绍1&email=user1%40email.com&code=" + code)
	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		t.Error(err)
		return
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		t.Error(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
		return
	}
	if string(body) != `{"Name":"user1","Email":"us***@email.com","SelfIntroduction":"自我介绍1"}` {
		t.Error()
		showDetail(req, res)
	}
}

func TestGetTokenByPassword(t *testing.T) {
	url := "http://127.0.0.1:10559/token"
	method := "POST"

	payload := strings.NewReader("username=user1&password=12345")

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		t.Error(err)
		return
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		t.Error(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
		return
	}

	tokenInfo, err := token_controller.GetTokenInfo(string(body))
	if err != nil {
		t.Error(err)
		return
	}
	if res.StatusCode != 200 {
		t.Error()
		showDetail(req, res)
		fmt.Printf("tokenInfo: %v\n", tokenInfo)
	}
}

func TestGetTokenByEmail(t *testing.T) {
	code, err := GetVerificationCode("user1@email.com")
	if err != nil {
		t.Error(err)
		return
	}

	url := "http://127.0.0.1:10559/token"
	method := "POST"

	payload := strings.NewReader("email=user1@email.com&code=" + code)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		t.Error(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
		return
	}

	tokenInfo, err := token_controller.GetTokenInfo(string(body))
	if err != nil {
		t.Error(err)
		return
	}
	if res.StatusCode != 200 {
		t.Error()
		showDetail(req, res)
		fmt.Printf("tokenInfo: %v\n", tokenInfo)
	}
}

func TestGetUserInfo(t *testing.T) {
	url := "http://127.0.0.1:10559/users/user1"
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		t.Error(err)
		return
	}

	res, err := client.Do(req)
	if err != nil {
		t.Error(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
		return
	}
	if string(body) != `{"Name":"user1","SelfIntroduction":"自我介绍1"}` {
		t.Error()
		showDetail(req, res)
	}
}

func TestGetUserInfoWithToken(t *testing.T) {
	token, err := GetTokenByPassword("user1", "12345")
	if err != nil {
		t.Error(err)
		return
	}

	url := "http://127.0.0.1:10559/users/user1"
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Error(err)
		return
	}
	req.Header.Add("Token", token)

	res, err := client.Do(req)
	if err != nil {
		t.Error(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
		return
	}
	if string(body) != `{"Name":"user1","Email":"us***@email.com","SelfIntroduction":"自我介绍1"}` {
		t.Error()
		showDetail(req, res)
	}
}

func TestUpdateUserInfo(t *testing.T) {
	token, err := GetTokenByPassword("user1", "12345")
	if err != nil {
		t.Error(err)
		return
	}

	url := "http://127.0.0.1:10559/users/user1"
	method := "PATCH"

	payload := strings.NewReader("username=user1_&selfIntroduction=自我介绍1。")

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		t.Error(err)
		return
	}
	req.Header.Add("Token", token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		t.Error(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
		return
	}
	if string(body) != `{"Name":"user1_","Email":"us***@email.com","SelfIntroduction":"自我介绍1。"}` {
		t.Error()
		showDetail(req, res)
	}
}

func TestUpdateUserPassword(t *testing.T) {
	token, err := GetTokenByPassword("user1_", "12345")
	if err != nil {
		t.Error(err)
		return
	}
	code, err := GetVerificationCode("user1@email.com")
	if err != nil {
		t.Error(err)
		return
	}

	url := "http://127.0.0.1:10559/users/user1_"
	method := "PATCH"

	payload := strings.NewReader("password=123456&oldEmailCode=" + code)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		t.Error(err)
		return
	}
	req.Header.Add("Token", token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		t.Error(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
		return
	}
	if string(body) != `{"Name":"user1_","Email":"us***@email.com","SelfIntroduction":"自我介绍1。"}` {
		t.Error()
		showDetail(req, res)
	}
}

func TestUpdateUserEmail(t *testing.T) {
	token, err := GetTokenByPassword("user1_", "123456")
	if err != nil {
		t.Error(err)
		return
	}
	oldEmailCode, err := GetVerificationCode("user1@email.com")
	if err != nil {
		t.Error(err)
		return
	}
	newEmailCode, err := GetVerificationCode("user1@email2.com")
	if err != nil {
		t.Error(err)
		return
	}

	url := "http://127.0.0.1:10559/users/user1_"
	method := "PATCH"

	payload := strings.NewReader("email=user1%40email2.com&newEmailCode=" + newEmailCode + "&oldEmailCode=" + oldEmailCode)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		t.Error(err)
		return
	}
	req.Header.Add("Token", token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		t.Error(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
		return
	}
	if string(body) != `{"Name":"user1_","Email":"us***@email2.com","SelfIntroduction":"自我介绍1。"}` {
		t.Error()
		showDetail(req, res)
	}
}

func TestDeleteUser(t *testing.T) {
	token, err := GetTokenByPassword("user1_", "123456")
	if err != nil {
		t.Error(err)
		return
	}

	url := "http://127.0.0.1:10559/users/user1_"
	method := "DELETE"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		t.Error(err)
		return
	}
	req.Header.Add("Token", token)

	res, err := client.Do(req)
	if err != nil {
		t.Error(err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 204 {
		t.Error()
		showDetail(req, res)
	}
}
