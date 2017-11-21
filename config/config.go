package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)


func configGet(name string, data interface{}, defaultData interface{}) {

	//mux.Lock()

	//defer mux.Unlock()

	absPath, _ := filepath.Abs(fmt.Sprintf("configs/%s.json", name))

	file, err := os.Open(absPath)

	if err!=nil{
		//找上一级目录
		absPath, _ = filepath.Abs(fmt.Sprintf("../configs/%s.json", name))

		file, err = os.Open(absPath)

	}
	if err != nil {

		panic(fmt.Sprintf("open %s config file failed:%s", name, err.Error()))

		data = defaultData

	} else {

		defer file.Close()

		decoder := json.NewDecoder(file)

		errDecode := decoder.Decode(data)

		//if name == "cache" {
		//}
		if errDecode != nil {
			//记录日志
			fmt.Errorf(fmt.Sprintf("decode %s config error:%s", name, errDecode.Error()))
			data = defaultData
		}
	}
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
