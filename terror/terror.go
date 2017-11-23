package terror

import (
	"fmt"
	"github.com/tonyjt/tgo_v2/pconst"
)

type TError struct {
	Code      int
	Msg       string
	Level     Level
	MsgCustom string
}

type Level int8

const (
	LevelDefault   Level = iota
	LevelException       //异常
)

func New(code int) *TError {
	err := &TError{Code: code}
	if code < 100000 && code >= 10000 {
		err.Level = LevelException
	} else {
		err.Level = LevelDefault
	}
	return err
}

func NewFromError(err error) *TError {
	if err == nil {
		return nil
	}
	return &TError{Code: pconst.ERROR_SYSTEM, Msg: err.Error(), Level: LevelException}
}

func (p *TError) GetMsg() string {
	if p.MsgCustom == "" {
		return p.Msg
	}

	if p.Msg == "" {
		return p.MsgCustom
	}

	return fmt.Sprintf("%s:%s", p.Msg, p.MsgCustom)
}
func (p *TError) Error() string {
	return p.GetMsg()
}
