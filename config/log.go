package config

type Log struct {
	File       string
	MaxSize    int //mb
	MaxBackups int
	MaxAge     int
	Compress   bool
	Level      uint32
}

var (
	logConfig *Log
)

func init() {
	logConfig = &Log{}

	err := configGet("log", logConfig, false, nil)

	if err != nil {
		defaultLogConfig := configLogGetDefault()
		logConfig = defaultLogConfig
	}

}

func configLogGetDefault() *Log {
	return &Log{File: "/data/code/tgo_v2.log", MaxSize: 500, MaxBackups: 100, MaxAge: 30}
}

func LogGet() *Log {
	if logConfig == nil {
		panic("log config is nil")
	}
	return logConfig
}
