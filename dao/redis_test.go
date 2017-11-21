package dao

import (
	"context"
	"testing"
)

type ModelRedisHello struct {
	HelloWord string
}

func TestRedis_Set(t *testing.T) {
	redis := NewRedisTest()

	err := redis.Set(context.Background(), "tonyjt", "s:35:\"h5_9989f070d21dc983cc7bbc5c6a013080\";")

	if err != nil {
		t.Error(err)
		return
	}
	var data string
	exists, err := redis.Get(context.Background(), "tonyjt", &data)

	if err != nil {
		t.Error(err)
	} else if !exists {
		t.Error("not exists")
	}

}

func TestRedis_Get(t *testing.T) {
	redis := NewRedisTest()

	var data string
	exists, err := redis.Get(context.Background(), "tonyjt", &data)

	if err != nil {
		t.Error(err)
	} else if !exists {
		t.Error("result false")
	}
}

func TestRedis_SetEx(t *testing.T) {
	redis := NewRedisTest()

	err := redis.SetEx(context.Background(), "setex", "asdfsdf", 60)

	if err != nil {
		t.Error("result false")
	}
}

func TestRedis_Incr(t *testing.T) {
	redis := NewRedisTest()

	_, err := redis.Incr(context.Background(), "incr")

	if err != nil {
		t.Error(err)
	}
}

func TestRedis_HSet(t *testing.T) {
	redis := NewRedisTest()

	err := redis.HSet(context.Background(), "hset", "k1", "sdfsdf")

	if err != nil {
		t.Error(err)
	}
}

func TestRedis_HSetNX(t *testing.T) {
	redis := NewRedisTest()

	_, err := redis.HSetNX(context.Background(), "hsetnx", "h1", "123123")

	if err != nil {
		t.Error(err)
	}
}

func TestRedis_Del(t *testing.T) {
	redis := NewRedisTest()

	err := redis.Del(context.Background(), "tonyjt")
	if err != nil {
		t.Error(err)
	}
}
func TestRedis_HIncrby(t *testing.T) {
	redis := NewRedisTest()

	_, err := redis.HIncrby(context.Background(), "hincr", "1", 1)

	if err != nil {
		t.Error("result false")
	}
}

func TestRedis_HMSet(t *testing.T) {
	redis := NewRedisTest()

	datas := make(map[string]ModelRedisHello)

	datas["1"] = ModelRedisHello{HelloWord: "HelloWord1"}
	datas["2"] = ModelRedisHello{HelloWord: "HelloWord2"}

	err := redis.HMSet("hmset1", datas)

	if err != nil {
		t.Error(err)
	}
}

func TestRedis_ZAddM(t *testing.T) {
	redis := NewRedisTest()

	datas := make(map[string]ModelRedisHello)

	datas["3"] = ModelRedisHello{HelloWord: "HelloWord3"}
	datas["2"] = ModelRedisHello{HelloWord: "HelloWord2"}
	datas["1"] = ModelRedisHello{HelloWord: "HelloWord1"}

	err := redis.ZAddM("zaddm1", datas)

	if err != nil {
		t.Errorf("result false")
	}
}

func TestRedis_ZRem(t *testing.T) {
	redis := NewRedisTest()

	err := redis.ZRem("zaddm1", 2, 3)

	if err != nil {
		t.Errorf("result false")
	}
}
func TestRedis_MSet(t *testing.T) {
	redis := NewRedisTest()
	value := make(map[string]ModelRedisHello)
	value["mset1"] = ModelRedisHello{HelloWord: "1"}
	value["mset2"] = ModelRedisHello{HelloWord: "2"}
	value["mset3"] = ModelRedisHello{HelloWord: "3"}
	err := redis.MSet(value)
	if err != nil {
		t.Error("result false")
	}
}

func TestRedis_MGet(t *testing.T) {
	redis := NewRedisTest()

	value, err := redis.MGet("mset1", "mset2", "mset4", "mset3")

	if err != nil {
		t.Errorf("result false:%s", err.Error())
	} else if len(value) != 4 {
		t.Errorf("len is < 4:%d", len(value))
	}

}

func TestRedis_HDel(t *testing.T) {
	redis := NewRedisTest()

	key := "hmset1"
	err := redis.HDel(context.Background(), key, "1", "2")

	if err != nil {
		t.Error(err)
	}
}
func TestRedis_HMGet(t *testing.T) {
	redis := NewRedisTest()

	data, err := redis.HMGet("hmset1", "1", "2", "3")

	if err != nil {
		t.Errorf("result false:%s", err.Error())
	} else if len(data) != 3 {
		t.Errorf("len is lt :%d", len(data))
	}
}

func TestRedis_ZRevRange(t *testing.T) {
	redis := NewRedisTest()

	_, err := redis.ZRevRange("zaddm1", 0, 1)

	if err != nil {
		t.Errorf("result false:%s", err.Error())
	}
}

type TestRedis struct {
	Redis
}

func NewRedisTest() *TestRedis {

	componentDao := &TestRedis{Redis{Key: "test"}}

	return componentDao
}

func (c *TestRedis) Set(ctx context.Context, name string, key string) (err error) {

	return c.Redis.Set(ctx, name, key)
}

func (c *TestRedis) Get(ctx context.Context, name string, key *string) (exists bool, err error) {
	return c.Redis.Get(ctx, name, key)
}
func (c *TestRedis) MGet(keys ...string) ([]*ModelRedisHello, error) {

	var datas []*ModelRedisHello

	err := c.Redis.MGet(context.Background(), keys, &datas)

	return datas, err
}

func (c *TestRedis) HMSet(key string, value map[string]ModelRedisHello) (err error) {
	datas := make(map[string]interface{})

	for k, v := range value {
		datas[k] = v
	}
	err = c.Redis.HMSet(context.Background(), key, datas)
	return
}

func (c *TestRedis) HMGet(key string, fields ...string) ([]*ModelRedisHello, error) {
	var datas []*ModelRedisHello

	var args []interface{}

	for _, item := range fields {
		args = append(args, item)
		//datas = append(datas, &ModelRedisHello{})
	}
	err := c.Redis.HMGet(context.Background(), key, args, &datas)

	return datas, err
}

func (c *TestRedis) ZAddM(key string, value map[string]ModelRedisHello) error {
	datas := make(map[string]interface{})

	for k, v := range value {
		datas[k] = v
	}
	return c.Redis.ZAddM(context.Background(), key, datas)
}

func (c *TestRedis) ZRem(key string, data ...interface{}) error {

	return c.Redis.ZRem(context.Background(), key, data...)
}

func (c *TestRedis) ZRevRange(key string, start int, end int) ([]ModelRedisHello, error) {

	var data []*ModelRedisHello

	err := c.Redis.ZRevRange(context.Background(), key, start, end, &data)
	var value []ModelRedisHello
	if err == nil {

		for _, item := range data {
			if item != nil {
				value = append(value, *item)
			} else {
				value = append(value, ModelRedisHello{})
			}
		}
	}
	return value, err
}

func (c *TestRedis) MSet(value map[string]ModelRedisHello) error {
	datas := make(map[string]interface{})

	for k, v := range value {
		datas[k] = v
	}
	return c.Redis.MSet(context.Background(), datas)
}
