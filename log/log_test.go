package log

import "testing"

func TestLog(t *testing.T) {
	Log(LevelDebug, "debug")
	Log(LevelInfo, "info")
	Log(LevelWarn, "warn")
	Log(LevelError, "error")
}

func BenchmarkLog(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Log(LevelInfo, "info")
	}
}
