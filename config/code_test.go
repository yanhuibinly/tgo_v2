package config

import (
	"testing"
)

func TestCodeGetMsg(t *testing.T) {

	var code int

	code = 1001

	msg := CodeGetMsg(code)

	if msg == "unknown error" {
		t.Error("unknown error")
	}
}

func BenchmarkCodeGetMsg(b *testing.B) {

	for i := 0; i < b.N; i++ {
		code := 1001

		CodeGetMsg(code)
	}
}
