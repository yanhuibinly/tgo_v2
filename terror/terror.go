package terror

import (
	"fmt"
	"github.com/tonyjt/tgo_v2/pconst"
	"strconv"
)

type TError struct{
	Code int
	Msg string
	MsgCustom string
}

func New(code int)*TError{
	return &TError{Code:code}
}

func NewFromError(err error) *TError{
	if err==nil{
		return nil
	}
	return &TError{Code:pconst.ERROR_SYSTEM, Msg:err.Error()}
}

func (p *TError) GetMsg()string{
	if p.MsgCustom == ""{
		return p.Msg
	}

	if p.Msg ==""{
		return p.MsgCustom
	}

	return fmt.Sprintf("%s:%s",p.Msg,p.MsgCustom)
}
func (p *TError) Error() string{
	return strconv.Itoa(p.Code)
}