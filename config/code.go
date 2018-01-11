package config

type Code struct {
	Public  map[int]string
	Private map[int]string
}

var (
	codeConfig *Code
)

func init() {
	codeConfig = &Code{}
	codeConfig.Private = make(map[int]string)

	codeConfig.Public = make(map[int]string)

	err := configGet("code_private", &codeConfig.Private)

	if err != nil {
		codeConfig.Private = configCodeGetDefaultPrivate()
	}
	err = configGet("code_public", &codeConfig.Public)
	if err != nil {
		codeConfig.Public = configCodeGetDefaultPublic()
	}
}

// CodeGetMsg 获取message
func CodeGetMsg(code int) string {

	var msg string

	msg, ok := codeConfig.Private[code]

	if !ok {
		msg, ok = codeConfig.Public[code]
		if !ok {
			msg = "unknown error"
		}
	}

	return msg
}

func configCodeGetDefaultPrivate() map[int]string {
	return map[int]string{1001: "success"}
}

func configCodeGetDefaultPublic() map[int]string {
	return map[int]string{0: "success"}
}
