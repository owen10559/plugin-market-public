package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

var Config map[string]map[string]string

func init() {
	f, err := os.Open("config.json")
	if err != nil {
		fmt.Println(err)
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Println(err)
	}
	err = f.Close()
	if err != nil {
		fmt.Println(err)
	}
	err = json.Unmarshal(b, &Config)
	if err != nil {
		fmt.Println(err)
	}
}
