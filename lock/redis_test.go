package lock

import (
	"context"
	"testing"
)

func TestRedis(t *testing.T) {

	ctx := context.Background()

	key := "testRedis"

	mutex := RedisGet(ctx, key)

	err := RedisLock(ctx, mutex)

	if err != nil {
		t.Error(err)
	}

	r := RedisUnlock(ctx, mutex)

	if !r {
		t.Error("unlock")
	}
}
