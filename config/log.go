package config

type Log struct{
	File string
	MaxSize int		//mb
	MaxBackups int
	MaxAge int
	Compress bool
	Level uint32
}

var (
	logConfig *Log
)


func init() {
	logConfig = &Log{}

	defaultLogConfig := configLogGetDefault()


	configGet("log", logConfig, defaultLogConfig)

}

func configLogGetDefault() *Log {
	return &Log{File:"/data/code/tgo_v2.log",MaxSize:500,MaxBackups:100,MaxAge:30}
}

func LogGet()(*Log){

	return logConfig
}


