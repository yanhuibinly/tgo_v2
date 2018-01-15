package config

import "sync"

type Code struct {
	Public  map[int]string
	Private map[int]string
}

var (
	codeConfig   *Code
	mutexCodePri *sync.RWMutex
	mutexCodePub *sync.RWMutex
)

func init() {
	codeConfig = &Code{}
	codeConfig.Private = make(map[int]string)
	codeConfig.Public = make(map[int]string)
	mutexCodePri = new(sync.RWMutex)
	mutexCodePub = new(sync.RWMutex)
	err := configGet("code_private", &codeConfig.Private, true, mutexCodePri)

	if err != nil {
		codeConfig.Private = configCodeGetDefaultPrivate()
	}
	err = configGet("code_public", &codeConfig.Public, true, mutexCodePub)
	if err != nil {
		codeConfig.Public = configCodeGetDefaultPublic()
	}
}

// CodeGetMsg 获取message
func CodeGetMsg(code int) string {

	var msg string

	mutexCodePri.RLock()
	msg, ok := codeConfig.Private[code]
	defer mutexCodePri.RUnlock()
	if !ok {
		mutexCodePub.RLock()
		msg, ok = codeConfig.Public[code]
		defer mutexCodePub.RUnlock()
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
