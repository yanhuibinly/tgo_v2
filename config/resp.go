package config

type Resp struct {
	Code string
	Msg  string
	Data string
}

var (
	respConfig *Resp
)

func init() {
	respConfig = &Resp{}

	err := configGet("resp", respConfig)

	if err != nil {
		defaultConfig := configRespGetDefault()

		respConfig = defaultConfig
	}

	return
}

func configRespGetDefault() *Resp {
	return &Resp{Code: "code", Msg: "msg", Data: "data"}
}

func RespGet() *Resp {
	return respConfig
}
