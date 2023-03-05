package apis_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type user struct {
	username         string
	password         string
	selfIntroduction string
	email            string
	token            string
}

var authors = []user{
	{"author1", "author1_passwd", "插件作者1", "author1@email.com", ""},
	{"author2", "author2_passwd", "插件作者2", "author2@email.com", ""},
}

type plugin struct {
	name        string
	description string
}

var plugins = []plugin{
	{"plugin_demo1", "插件1"},
	{"plugin_demo2", "插件2"},
	{"plugin_demo3", "插件3"},
}

func TestAddPluginAuthors(t *testing.T) {

	for i, v := range authors {
		t.Run(v.username, func(t *testing.T) {
			code, err := GetVerificationCode(v.email)
			if err != nil {
				t.Error(err)
				return
			}

			url := "http://127.0.0.1:10559/users"
			method := "POST"

			payload := strings.NewReader("username=" + v.username + "&password=" + v.password + "&selfIntroduction=" + v.selfIntroduction + "&email=" + v.email + "&code=" + code)
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

			if string(body) != `{"Name":"`+v.username+`","Email":"au***@email.com","SelfIntroduction":"`+v.selfIntroduction+`"}` {
				t.Error()
				showDetail(req, res, body)
			}

			authors[i].token, err = GetTokenByPassword(v.username, v.password)
			if err != nil {
				t.Error(err)
				return
			}
		})
	}

	// user2Token, err = GetTokenByPassword("user2", "22345")
	// if err != nil {
	// 	t.Error(err)
	// 	return
	// }
	// user3Token, err = GetTokenByPassword("user3", "33345")
	// if err != nil {
	// 	t.Error(err)
	// 	return
	// }
}

func TestAddPlugins(t *testing.T) {

	for i, plugin := range plugins {
		var author user
		if i <= 1 {
			author = authors[0]
		} else {
			author = authors[1]
		}
		t.Run(author.username+"/"+plugin.name, func(t *testing.T) {
			url := "http://127.0.0.1:10559/plugins/" + author.username
			method := "POST"

			payload := &bytes.Buffer{}
			writer := multipart.NewWriter(payload)
			file, err := os.Open("../testdata/plugins/" + plugin.name + ".dll")
			if err != nil {
				t.Error(err)
				return
			}
			defer file.Close()

			part1, errFile1 := writer.CreateFormFile("plugin", filepath.Base("../testdata/plugins/"+plugin.name+".dll"))
			_, errFile1 = io.Copy(part1, file)
			if errFile1 != nil {
				t.Error(err)
				return
			}
			_ = writer.WriteField("description", plugin.description)
			err = writer.Close()
			if err != nil {
				t.Error(err)
				return
			}

			client := &http.Client{}
			req, err := http.NewRequest(method, url, payload)

			if err != nil {
				t.Error(err)
				return
			}
			req.Header.Add("Token", author.token)

			req.Header.Set("Content-Type", writer.FormDataContentType())
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
			if string(body) != `{"Name":"`+plugin.name+`","Author":"`+author.username+`","Description":"`+plugin.description+`","DownloadCount":0}` {
				t.Error()
				showDetail(req, res, body)
			}
		})
	}
}

func TestSearchPluginsByPluginName(t *testing.T) {
	url := "http://127.0.0.1:10559/plugins?plugin_name=plugin_demo"
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
	if string(body) != `{"plugins":[{"Name":"plugin_demo1","Author":"author1","Description":"插件1","DownloadCount":0},{"Name":"plugin_demo2","Author":"author1","Description":"插件2","DownloadCount":0},{"Name":"plugin_demo3","Author":"author2","Description":"插件3","DownloadCount":0}]}` {
		t.Error()
		showDetail(req, res, body)
	}
}

func TestSearchPluginsByAuthorName(t *testing.T) {
	url := "http://127.0.0.1:10559/plugins?author_name=author1"
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		fmt.Println(err)
		return
	}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	if string(body) != `{"plugins":[{"Name":"plugin_demo1","Author":"author1","Description":"插件1","DownloadCount":0},{"Name":"plugin_demo2","Author":"author1","Description":"插件2","DownloadCount":0}]}` {
		t.Error()
		showDetail(req, res, body)
	}
}

func TestSearchPluginsByPluginNameAndAuthorName(t *testing.T) {
	url := "http://127.0.0.1:10559/plugins?plugin_name=plugin_demo&author_name=author1"
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		fmt.Println(err)
		return
	}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	if string(body) != `{"plugins":[{"Name":"plugin_demo1","Author":"author1","Description":"插件1","DownloadCount":0},{"Name":"plugin_demo2","Author":"author1","Description":"插件2","DownloadCount":0}]}` {
		t.Error()
		showDetail(req, res, body)
	}
}

func TestGetPluginsInfo(t *testing.T) {
	url := "http://127.0.0.1:10559/plugins/" + authors[0].username + "/" + plugins[0].name
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
	if string(body) != fmt.Sprintf(`{"Name":"%s","Author":"%s","Description":"%s","DownloadCount":0}`, plugins[0].name, authors[0].username, plugins[0].description) {
		t.Error()
		showDetail(req, res)
	}
}

func TestDownloadPlugin(t *testing.T) {
	url := "http://127.0.0.1:10559/plugins/" + authors[0].username + "/" + plugins[0].name + "/download"
	method := "GET"

	payload := strings.NewReader("")

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

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

	content, err := os.ReadFile("../testdata/plugins/" + plugins[0].name + ".dll")
	if err != nil {
		t.Error(err)
		return
	}

	if string(body) != string(content) {
		t.Errorf("expectd: %s", string(content))
		showDetail(req, res)
	}
}

func TestPluginDownloadCount(t *testing.T) {
	url := "http://127.0.0.1:10559/plugins/" + authors[0].username + "/" + plugins[0].name
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
	if string(body) != fmt.Sprintf(`{"Name":"%s","Author":"%s","Description":"%s","DownloadCount":1}`, plugins[0].name, authors[0].username, plugins[0].description) {
		t.Error()
		showDetail(req, res, body)
	}
}

func TestUpdatePluginInfo(t *testing.T) {

	url := "http://127.0.0.1:10559/plugins/" + authors[0].username + "/" + plugins[0].name
	method := "PATCH"

	plugins[0].name = "plugin_demo1v2"
	plugins[0].description = "插件1v2"

	payload := strings.NewReader(fmt.Sprintf("description=%s&name=%s", plugins[0].description, plugins[0].name))

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		t.Error(err)
		return
	}
	req.Header.Add("Token", authors[0].token)
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

	if string(body) != fmt.Sprintf(`{"Name":"%s","Author":"%s","Description":"%s","DownloadCount":1}`, plugins[0].name, authors[0].username, plugins[0].description) {
		t.Error()
		showDetail(req, res, body)
	}
}

func TestDownloadPluginAfterUpdatePluginInfo(t *testing.T) {
	url := "http://127.0.0.1:10559/plugins/" + authors[0].username + "/" + plugins[0].name + "/download"
	method := "GET"

	payload := strings.NewReader("")

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

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

	content, err := os.ReadFile("../testdata/plugins/" + plugins[0].name + ".dll")
	if err != nil {
		t.Error(err)
		return
	}

	if string(body) != string(content) {
		t.Errorf("expectd: %s", string(content))
		showDetail(req, res)
	}
}

func TestPluginDownloadCountAfterUpdatePluginInfo(t *testing.T) {
	url := "http://127.0.0.1:10559/plugins/" + authors[0].username + "/" + plugins[0].name
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
	if string(body) != fmt.Sprintf(`{"Name":"%s","Author":"%s","Description":"%s","DownloadCount":2}`, plugins[0].name, authors[0].username, plugins[0].description) {
		t.Error()
		showDetail(req, res, body)
	}
}

func TestUpdatePluginFile(t *testing.T) {
	oldPlugin := plugins[1]
	author := authors[0]
	newPlugin := plugin{"plugin_demo2v2", "插件2v2"}
	plugins[1] = newPlugin

	// var newPlugin = plugin{"plugin_demo2", ""}

	url := "http://127.0.0.1:10559/plugins/" + author.username + "/" + oldPlugin.name
	method := "PATCH"

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	_ = writer.WriteField("description", newPlugin.description)

	file, err := os.Open("../testdata/plugins/" + newPlugin.name + ".dll")
	if err != nil {
		t.Error(err)
	}
	defer file.Close()
	part2, errFile2 := writer.CreateFormFile("plugin", filepath.Base("../testdata/plugins/"+newPlugin.name+".dll"))
	_, errFile2 = io.Copy(part2, file)
	if errFile2 != nil {
		fmt.Println(errFile2)
		return
	}
	err = writer.Close()
	if err != nil {
		fmt.Println(err)
		return
	}

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Token", author.token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	req.Header.Set("Content-Type", writer.FormDataContentType())
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	if string(body) != fmt.Sprintf(`{"Name":"%s","Author":"%s","Description":"%s","DownloadCount":0}`, newPlugin.name, author.username, newPlugin.description) {
		t.Error()
		showDetail(req, res, body)
	}
}

func TestDownloadPluginAfterUpdatePluginFile(t *testing.T) {
	author := authors[0]
	plugin := plugins[1]

	url := "http://127.0.0.1:10559/plugins/" + author.username + "/" + plugin.name + "/download"
	method := "GET"

	payload := strings.NewReader("")

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

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

	content, err := os.ReadFile("../testdata/plugins/" + plugin.name + ".dll")
	if err != nil {
		t.Error(err)
		return
	}

	if string(body) != string(content) {
		t.Errorf("expectd: %s", string(content))
		showDetail(req, res, body)
	}
}

func TestPluginDownloadCountAfterUpdatePluginFile(t *testing.T) {
	author := authors[0]
	plugin := plugins[1]

	url := "http://127.0.0.1:10559/plugins/" + author.username + "/" + plugin.name
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
	if string(body) != fmt.Sprintf(`{"Name":"%s","Author":"%s","Description":"%s","DownloadCount":1}`, plugin.name, author.username, plugin.description) {
		t.Error()
		showDetail(req, res, body)
	}
}

func TestDeletePlugin(t *testing.T) {
	author := authors[1]
	plugin := plugins[2]

	url := "http://127.0.0.1:10559/plugins/" + author.username + "/" + plugin.name
	method := "DELETE"

	payload := strings.NewReader("")

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Token", author.token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 204 {
		t.Error()
		showDetail(req, res)
	}
}

func TestGetPluginsInfoAfterDelete(t *testing.T) {
	author := authors[1]
	plugin := plugins[2]

	url := "http://127.0.0.1:10559/plugins/" + author.username + "/" + plugin.name
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

	if res.StatusCode != 404 {
		t.Error()
		showDetail(req, res)
	}
}
