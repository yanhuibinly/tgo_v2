package config

import (
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"os"
	"path/filepath"
	"sync"
)

func configGet(name string, data interface{}, sync bool, mutex *sync.RWMutex) (err error) {

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

		if sync && mutex != nil {
			mutex.Lock()
			defer mutex.Unlock()
		}
		err = decoder.Decode(data)

		//if name == "cache" {
		//}
		if err != nil {
			//记录日志
			fmt.Errorf(fmt.Sprintf("decode %s config error:%s", name, err.Error()))
		}
		if sync {
			go configSync(absPath, data, mutex)
		}

	}
	return
}

//configSync 检测文件修改,更新config
func configSync(path string, data interface{}, mutex *sync.RWMutex) {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Errorf(fmt.Sprintf("file watch failed:%s", err.Error()))
	}
	defer watcher.Close()

	err = watcher.Add(path)

	if err != nil {
		fmt.Printf("watcher add err:%s ", err.Error())
	}
	for {
		select {
		case event := <-watcher.Events:
			fmt.Println("event:", event)

			var file *os.File
			file, err = os.Open(path)

			if err != nil {
				fmt.Errorf("sync open file err: %s\n", err.Error())
			} else {
				mutex.Lock()
				err = configParseFile(file, data)
				mutex.Unlock()
				if err != nil {
					fmt.Errorf("sync config parse file err: %s\n", err.Error())
				}

				file.Close()
			}

		case err := <-watcher.Errors:
			fmt.Errorf("file watcher error: %s \n", err.Error())
		}
	}
}

func configParseFile(file *os.File, data interface{}) (err error) {

	decoder := json.NewDecoder(file)

	err = decoder.Decode(data)

	if err != nil {
		//记录日志
		fmt.Errorf("decode file error:%s \n", err.Error())
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
