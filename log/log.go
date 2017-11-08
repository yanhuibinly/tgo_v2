package log

import "fmt"

type Level int8

const(
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

func Log(level Level,msg interface{}) {
	fmt.Printf("%d:%s\n",level,msg)
}

func Logf(level Level,format string, msg ...interface{}) {

	fmt.Printf("%d:%s\n",level,fmt.Sprintf(format,msg...))
}


func Errorf(format string, msg ...interface{}){
	Logf(LevelError,format,msg...)
}

func Error(msg interface{}){
	Log(LevelError,msg)
}