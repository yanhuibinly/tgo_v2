package config

import "testing"

func TestCodeGetMsg(t *testing.T) {

	var code int

	code = 1001

	msg:= CodeGetMsg(code)

	if msg == "unknown error"{
		t.Error("unknown error")
	}
}