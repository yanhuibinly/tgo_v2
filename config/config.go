package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func configGet(name string, data interface{}) (err error) {

	//mux.Lock()

	//defer mux.Unlock()

	absPath, _ := filepath.Abs(fmt.Sprintf("configs/%s.json", name))

	file, err := os.Open(absPath)

	if err != nil {
		//找上一级目录
		absPath, _ = filepath.Abs(fmt.Sprintf("../configs/%s.json", name))

		file, err = os.Open(absPath)

	}
	if err != nil {
		if name != "resp" {
			panic(fmt.Sprintf("open %s config file failed:%s", name, err.Error()))
		}

	} else {

		defer file.Close()

		decoder := json.NewDecoder(file)

		err = decoder.Decode(data)

		//if name == "cache" {
		//}
		if err != nil {
			//记录日志
			fmt.Errorf(fmt.Sprintf("decode %s config error:%s", name, err.Error()))
		}
	}
	return
}

func configPathExist(name string) bool {
	path := fmt.Sprintf("configs/%s.json", name)
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

func ConfigReload() {

}
