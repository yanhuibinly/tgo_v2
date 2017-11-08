package config

import (
	"reflect"
	"strconv"
	"strings"
	"time"
	"github.com/tonyjt/tgo_v2/terror"
	"github.com/tonyjt/tgo_v2/pconst"
	"github.com/tonyjt/tgo_v2/log"
	"math/rand"
)

type App struct {
	Configs map[string]interface{}
}

var (

	appConfig *App
)


func init(){

		appConfig = &App{}

		defaultAppConfig := appGetDefault()

		configGet("app", appConfig, defaultAppConfig)

}



func appGetDefault() *App {
	return &App{map[string]interface{}{"Env": "idc", "UrlUserLogin": "http://user.haiziwang.com/user/CheckLogin"}}
}

func AppGet(key string) interface{} {

	config, exists := appConfig.Configs[key]

	if !exists {
		return nil
	}
	return config
}

func AppGetString(key string, defaultConfig string) string {

	config := AppGet(key)

	if config == nil {
		return defaultConfig
	} else {
		configStr := config.(string)

		if strings.Trim(configStr," ") == ""{
			configStr = defaultConfig
		}
		return configStr
	}
}

func AppFailoverGet(key string) (string, error) {

	var server string

	var err error

	failoverConfig := AppGet(key)

	if failoverConfig == nil {
		log.Errorf("config % is null",key)
		err = terror.New(pconst.ERROR_CONFIG_NULL)
	} else {

		failoverUrl := failoverConfig.(string)

		if strings.Trim(failoverUrl," ") == ""{
			log.Errorf("config % is null",key)
			err = terror.New(pconst.ERROR_CONFIG_NULL)
		} else {
			failoverArray := strings.Split(failoverUrl, ",")

			randomMax := len(failoverArray)
			if randomMax == 0 {
				log.Errorf("config % is empty",key)
				err = terror.New(pconst.ERROR_CONFIG_NULL)
			} else {
				var randomValue int
				if randomMax > 1 {

					rand.Seed(time.Now().UnixNano())

					randomValue = rand.Intn(randomMax)

				} else {
					randomValue = 0
				}
				server = failoverArray[randomValue]

			}
		}
	}
	return server, err
}

func AppEnvGet() string {
	strEnv := AppGetString("Env","dev")


	return strEnv
}

func AppEnvIsDev() bool {
	env := AppEnvGet()

	if env == "dev" || env == "debug" {
		return true
	}
	return false
}

func AppEnvIsBeta() bool {
	env := AppEnvGet()

	if env == "beta" {
		return true
	}
	return false
}

//AppGetSlice 获取slice配置，data必须是指针slice *[]，目前支持string,int,int64,bool,float64,float32
func AppGetSlice(key string, data interface{}) error {

	dataStrConfig := AppGetString(key, "")

	if strings.Trim(dataStrConfig," ") == "" {

		log.Errorf("config %s is empty",key)
		return terror.New(pconst.ERROR_CONFIG_NULL)
	}

	dataStrSlice := strings.Split(dataStrConfig, ",")

	dataType := reflect.ValueOf(data)

	//不是指针Slice
	if dataType.Kind() != reflect.Ptr || dataType.Elem().Kind() != reflect.Slice {

		log.Errorf("config %s is not pt or slice",key)
		return terror.New(pconst.ERRPR_CONFIG_SLICE)
	}

	dataSlice := dataType.Elem()

	//dataSlice = dataSlice.Slice(0, dataSlice.Cap())

	dataElem := dataSlice.Type().Elem()

	for _, dataStr := range dataStrSlice {

		if dataStrConfig ==""{
			continue
		}
		var errConv error
		var item interface{}

		switch dataElem.Kind() {
		case reflect.String:
			item = dataStr
		case reflect.Int:
			item, errConv = strconv.Atoi(dataStr)
		case reflect.Int64:
			item, errConv = strconv.ParseInt(dataStr, 10, 64)
		case reflect.Bool:
			item, errConv = strconv.ParseBool(dataStr)
		case reflect.Float64:
			item, errConv = strconv.ParseFloat(dataStr, 64)
		case reflect.Float32:
			var item64, errConv = strconv.ParseFloat(dataStr, 32)
			if errConv == nil {
				item = float32(item64)
			}
			/*
				case reflect.Struct:
					var de
					errConv = json.Unmarshal([]byte(dataStr), de.Interface())*/
		default:
			log.Errorf("type not support")
			return terror.New(pconst.ERRPR_CONFIG_SLICE_TYPE)
		}
		if errConv != nil {
			log.Errorf("convert config failed error:%s", errConv.Error())

			return terror.New(pconst.ERRPR_CONFIG_SLICE_CONVERT)
		}

		dataSlice.Set(reflect.Append(dataSlice, reflect.ValueOf(item)))
	}
	return nil
}
