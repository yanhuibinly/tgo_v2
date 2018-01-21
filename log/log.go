package log

import (
	"github.com/sirupsen/logrus"
	"github.com/tonyjt/tgo_v2/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

//Level level
type Level uint32

//const level
const (
	LevelPanic Level = iota
	LevelFatal
	LevelError
	LevelWarn
	LevelInfo
	LevelDebug
)

var (
	logger *logrus.Logger
)

func init() {
	conf := config.LogGet()
	if conf == nil {
		panic("log config file not found")
	}
	logger = logrus.StandardLogger()
	logger.Formatter = new(logrus.JSONFormatter)
	logger.Out = &lumberjack.Logger{
		Filename:   conf.File,
		MaxSize:    conf.MaxSize,
		MaxBackups: conf.MaxBackups,
		MaxAge:     conf.MaxAge,
		Compress:   conf.Compress}

	logger.SetLevel(logrus.Level(conf.Level))
}

//Log log
func Log(level Level, msg ...interface{}) {
	switch level {
	case LevelDebug:
		logger.Debug(msg...)
	case LevelInfo:
		logger.Info(msg...)
	case LevelWarn:
		logger.Warn(msg...)
	case LevelError:
		logger.Error(msg...)
	case LevelFatal:
		logger.Fatal(msg...)
	case LevelPanic:
		logger.Panic(msg...)
	}
}

//Logf logf
func Logf(level Level, format string, msg ...interface{}) {

	switch level {
	case LevelDebug:
		logger.Debugf(format, msg...)
	case LevelInfo:
		logger.Infof(format, msg...)
	case LevelWarn:
		logger.Warnf(format, msg...)
	case LevelError:
		logger.Errorf(format, msg...)
	case LevelFatal:
		logger.Fatalf(format, msg...)
	case LevelPanic:
		logger.Panicf(format, msg...)
	}
}

//Errorf errorf
func Errorf(format string, msg ...interface{}) {
	Logf(LevelError, format, msg...)
}

//Error error
func Error(msg interface{}) {
	Log(LevelError, msg)
}

//LogStruct struct for default log
type LogStruct struct {
}

//NewLog New LogStruct
func NewLog() *LogStruct {
	return &LogStruct{}
}

//Error error
func (p *LogStruct) Error(format string, a ...interface{}) {
	Errorf(format, a...)
}

//Info info
func (p *LogStruct) Info(format string, a ...interface{}) {
	Logf(LevelInfo, format, a...)
}
