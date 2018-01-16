package lock

import (
	"context"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/go-redsync/redsync"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/tonyjt/tgo_v2/config"
	"github.com/tonyjt/tgo_v2/log"
	"github.com/tonyjt/tgo_v2/pconst"
	"github.com/tonyjt/tgo_v2/terror"
	"math/rand"
	"time"
)

var (
	redisPools []redsync.Pool
)

func init() {

	if !config.FeatureRedis() {
		panic("redis feature disabled")
	}

	conf := config.RedisGet()

	redisPools = append(redisPools, &redis.Pool{
		MaxIdle:     conf.Persist.PoolMaxIdle,
		IdleTimeout: time.Duration(conf.Persist.PoolIdleTimeout) * time.Millisecond,
		Dial: func() (redis.Conn, error) {
			rand.Seed(time.Now().UnixNano())
			i := rand.Intn(len(conf.Persist.Address))
			return redis.Dial("tcp", conf.Persist.Address[i],
				redis.DialConnectTimeout(time.Duration(conf.Persist.ConnectTimeout)*time.Millisecond),
				redis.DialReadTimeout(time.Duration(conf.Persist.ReadTimeout)*time.Millisecond),
				redis.DialWriteTimeout(time.Duration(conf.Persist.WriteTimeout)*time.Millisecond))
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	})
}

//RedisGet 获取redis lock
func RedisGet(ctx context.Context, key string) (mutex *redsync.Mutex) {

	rs := redsync.New(redisPools)

	mutex = rs.NewMutex(key)

	return
}

//RedisLock lock
func RedisLock(ctx context.Context, mutex *redsync.Mutex) (err error) {
	span, ctx := redisZipkinNewSpan(ctx, "lock")
	if span != nil {
		defer span.Finish()
	}
	err = mutex.Lock()

	if err != nil {
		err = redisProcessError(span, err, pconst.ERROR_LOCK_REDIS_LOCK, "lock failed")
	}
	return
}

//RedisUnlock unlock
func RedisUnlock(ctx context.Context, mutex *redsync.Mutex) bool {
	span, ctx := redisZipkinNewSpan(ctx, "unlock")
	if span != nil {
		defer span.Finish()
	}
	return mutex.Unlock()
}

func redisZipkinNewSpan(ctx context.Context, name string) (opentracing.Span, context.Context) {

	if config.FeatureZipkin() {
		return opentracing.StartSpanFromContext(ctx, fmt.Sprintf("lock:redis:%s", name))
	} else {
		return nil, ctx
	}
}

func redisProcessError(span opentracing.Span, err error, code int, formatter string, a ...interface{}) error {

	if err == nil {
		return err
	}

	log.Errorf("redis lock error :%s", fmt.Sprintf(formatter, a...))

	if span != nil {
		ext.Error.Set(span, true)
		span.SetTag("err", err)
	}

	terr := terror.New(code)

	return terr
}
