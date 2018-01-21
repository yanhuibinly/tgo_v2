package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/tonyjt/tgo_v2/config"
	"github.com/tonyjt/tgo_v2/log"
	"github.com/tonyjt/tgo_v2/pconst"
	"github.com/tonyjt/tgo_v2/terror"
	"github.com/youtube/vitess/go/pools"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	unpersist *pools.ResourcePool
	persist   *pools.ResourcePool // 持久化Pool
)

//ResourceConn ResourceConn
type ResourceConn struct {
	redis.Conn
	serverIndex int
}

func init() {
	if config.FeatureRedis() {
		//非持久化pool
		conf := config.RedisGet()
		unpersist = initRedisPool(conf.Unpersist)
		persist = initRedisPool(conf.Persist)
	}
}

func initRedisPool(conf config.RedisBase) *pools.ResourcePool {
	return pools.NewResourcePool(func() (pools.Resource, error) {
		c, serverIndex, err := dial(0, conf)
		return ResourceConn{Conn: c, serverIndex: serverIndex}, err
	}, conf.PoolMinActive, conf.PoolMaxActive, time.Duration(conf.PoolIdleTimeout)*time.Millisecond)

}
func dial(fromIndex int, config config.RedisBase) (conn redis.Conn, index int, err error) {

	if len(config.Address) > 0 {
		if fromIndex+1 > len(config.Address) {
			fromIndex = 0
		}

		for i, addr := range config.Address {
			if i >= fromIndex {
				conn, err = redis.Dial("tcp", addr,
					redis.DialConnectTimeout(time.Duration(config.ConnectTimeout)*time.Millisecond),
					redis.DialReadTimeout(time.Duration(config.ReadTimeout)*time.Millisecond),
					redis.DialWriteTimeout(time.Duration(config.WriteTimeout)*time.Millisecond))
				if err != nil {
					log.Errorf("dail redis pool error: %s", err.Error())
				} else {
					index = i
					return
				}
			}
		}
		return
	} else {
		err = terror.New(pconst.ERROR_REDIS_INIT_ADDRESS)
		return
	}
}

//Close close conn
func (r ResourceConn) Close() {
	r.Conn.Close()
}

//Redis redis struct
type Redis struct {
	Key        string
	Persistent bool
}

//ZipkinNewSpan new zipkin span for redis
func (p *Redis) ZipkinNewSpan(ctx context.Context, name string) (opentracing.Span, context.Context) {

	if config.FeatureZipkin() {
		return opentracing.StartSpanFromContext(ctx, fmt.Sprintf("redis:%s:%s", name, p.Key))
	} else {
		return nil, ctx
	}
}

//GetConn get redis conn
func (p *Redis) GetConn(ctx context.Context) (conn pools.Resource, err error) {

	span, ctx := p.ZipkinNewSpan(ctx, "conn")
	if span != nil {
		defer span.Finish()
	}

	var pool *pools.ResourcePool

	if p.Persistent {
		pool = persist
	} else {
		pool = unpersist
	}

	if pool == nil {
		log.Logf(log.LevelFatal, "redis pool is null")
		err = terror.New(pconst.ERROR_REDIS_POOL_NULL)
		if span != nil {
			span.SetTag("err:pool", err)
		}
		return
	}

	r, err := pool.Get(ctx)

	if err != nil {
		log.Errorf("redis get connection err:%s", err.Error())
		err = terror.New(pconst.ERROR_REDIS_POOL_GET)
		if span != nil {
			span.SetTag("err:pool", err)
		}
		return
	}

	if r == nil {
		log.Error("redis pool resource is null")
		err = terror.New(pconst.ERROR_REDIS_POOL_EMPTY)
		if span != nil {
			span.SetTag("err:pool", err)
		}
		return
	}

	rc := r.(ResourceConn)

	if rc.Conn.Err() != nil {
		log.Errorf("redis rc connection err:%s,serverIndex:%d", rc.Conn.Err().Error(), rc.serverIndex)

		rc.Close()
		//连接断开，重新打开
		var c redis.Conn
		var serverIndex int
		var conf config.RedisBase

		if p.Persistent {
			conf = config.RedisGet().Persist
		} else {
			conf = config.RedisGet().Unpersist
		}
		c, serverIndex, err = dial(rc.serverIndex+1, conf)
		if err != nil {
			pool.Put(r)
			log.Errorf("redis redail connection err:%s", err.Error())
			err = terror.New(pconst.ERROR_REDIS_POOL_REDIAL)
			if span != nil {
				span.SetTag("err:dial", err)
			}
			return
		} else {
			conn = ResourceConn{Conn: c, serverIndex: serverIndex}
			return
		}
	}

	conn = r
	return
}

//PutConn put back conn to pool
func (p *Redis) PutConn(ctx context.Context, resource pools.Resource) {
	var pool *pools.ResourcePool

	if p.Persistent {
		pool = persist
	} else {
		pool = unpersist
	}

	pool.Put(resource)
}

func (p *Redis) getKey(key string) string {

	conf := config.RedisGetBase(p.Persistent)

	prefixRedis := conf.Prefix

	if strings.Trim(key, " ") == "" {
		return fmt.Sprintf("%s:%s", prefixRedis, p.Key)
	}
	return fmt.Sprintf("%s:%s:%s", prefixRedis, p.Key, key)
}

//Do run redis command
func (p *Redis) Do(ctx context.Context, cmd string, args ...interface{}) (reply interface{}, err error) {
	span, ctx := p.ZipkinNewSpan(ctx, cmd)
	if span != nil {
		defer span.Finish()
	}

	redisResource, err := p.GetConn(ctx)

	if err != nil {
		return
	}

	defer p.PutConn(ctx, redisResource)

	redisClient := redisResource.(ResourceConn)

	var errDo error
	reply, errDo = redisClient.Do(cmd, args...)

	if errDo != nil {
		log.Errorf("run redis command %s failed:error:%s,args:%v", cmd, errDo.Error(), args)

		err = terror.New(pconst.ERROR_REDIS_DO)
		p.ZipkinTag(span, "do"+cmd, err)
	}
	return
}

// PipeDo
func (p *Redis) PipeDo(ctx context.Context, cmd string, args [][]interface{}, value []interface{}) (err error) {
	span, ctx := p.ZipkinNewSpan(ctx, cmd)
	if span != nil {
		defer span.Finish()
	}

	redisResource, err := p.GetConn(ctx)

	if err != nil {
		return
	}

	defer p.PutConn(ctx, redisResource)

	redisClient := redisResource.(ResourceConn)

	for _, v := range args {
		if err = redisClient.Send(cmd, v...); err != nil {
			log.Errorf("Send(%v) returned error %v", v, err)
			err = terror.New(pconst.ERROR_REDIS_PIPE_SEND)
			p.ZipkinTag(span, "send", err)
			return
		}
	}
	if err = redisClient.Flush(); err != nil {
		log.Errorf("Flush() returned error %v", err)
		err = terror.New(pconst.ERROR_REDIS_PIPE_FLUSH)
		p.ZipkinTag(span, "flush", err)
		return
	}
	for k, v := range args {
		var result interface{}
		result, err = redisClient.Receive()
		if err != nil {
			log.Errorf("Receive(%v) returned error %v", v, err)
			err = terror.New(pconst.ERROR_REDIS_PIPE_RECEIVE)
			p.ZipkinTag(span, "receive", err)
			return
		}
		if result == nil {
			value[k] = nil
			continue
		}
		if reflect.TypeOf(result).Kind() == reflect.Slice {

			byteResult := (result.([]byte))
			strResult := string(byteResult)

			if strResult == "[]" {
				value[k] = nil
				continue
			}
		}

		errorJson := json.Unmarshal(result.([]byte), value[k])

		if errorJson != nil {
			log.Errorf("get %s command result failed:%s", cmd, errorJson.Error())
			err = terror.New(pconst.ERROR_REDIS_PIPE_UNMARSHAL)
			p.ZipkinTag(span, "unmarshal", err)
			return
		}
	}

	return
}

// doSet
func (p *Redis) doSet(ctx context.Context, cmd string, key string, value interface{}, expire int, fields ...string) (reply interface{}, err error) {

	key = p.getKey(key)

	data, errJson := json.Marshal(value)

	if errJson != nil {

		log.Errorf("redis %s marshal data to json:%s", cmd, errJson.Error())

		err = terror.New(pconst.ERROR_REDIS_SET_MARSHAL)

		return
	}

	if expire == 0 {
		cacheConfig := config.RedisGetBase(p.Persistent)

		expire = cacheConfig.Expire
	}

	if len(fields) == 0 {
		if expire > 0 && strings.ToUpper(cmd) == "SET" {
			reply, err = p.Do(ctx, cmd, key, data, "ex", expire)
		} else {
			reply, err = p.Do(ctx, cmd, key, data)
		}

	} else {
		field := fields[0]

		reply, err = p.Do(ctx, cmd, key, field, data)

	}

	if err != nil {
		return
	}
	//set expire
	if expire > 0 && strings.ToUpper(cmd) != "SET" {
		p.Do(ctx, "EXPIRE", key, expire)
	}

	return
}

// doSetNX
func (p *Redis) doSetNX(ctx context.Context, cmd string, key string, value interface{}, expire int, field ...string) (exists bool, err error) {

	reply, err := p.doSet(ctx, cmd, key, value, expire, field...)

	if err != nil {
		return
	}

	row, ok := reply.(int64)

	if !ok {
		log.Errorf("HSetNX reply to int failed,key:%s,field:%s", key, field)
		err = terror.New(pconst.ERROR_REDIS_SETNX_REPLY)

		return
	}

	if row == 0 {
		exists = true
	} else {
		exists = false
	}

	return
}

// doMSet
func (p *Redis) doMSet(ctx context.Context, cmd string, key string, value map[string]interface{}) (reply interface{}, err error) {

	var args []interface{}

	if key != "" {
		key = p.getKey(key)
		args = append(args, key)
	}

	for k, v := range value {
		data, errJson := json.Marshal(v)

		if errJson != nil {
			log.Errorf("redis %s marshal data: %v to json:%s", cmd, v, errJson.Error())
			err = terror.New(pconst.ERROR_REDIS_MSET_MARSHAL)
			return
		}
		if key == "" {
			args = append(args, p.getKey(k), data)
		} else {
			args = append(args, k, data)
		}

	}
	/*
		if expire == 0 {
			cacheConfig := ConfigCacheGetRedisWithConn(p.Persistent)()

			expire = cacheConfig.Expire
		}

		if expire > 0 {
			args = append(args, "ex", expire)
		}*/

	reply, err = p.Do(ctx, cmd, args...)

	return
}

// doGet
func (p *Redis) doGet(ctx context.Context, cmd string, key string, value interface{}, fields ...string) (exists bool, err error) {

	key = p.getKey(key)

	var reply interface{}

	var args []interface{}

	args = append(args, key)

	for _, f := range fields {
		args = append(args, f)
	}

	reply, err = p.Do(ctx, cmd, args...)

	if err != nil {

		return
	}

	if reply == nil {
		return
	}

	if reflect.TypeOf(reply).Kind() == reflect.Slice {

		byteResult := (reply.([]byte))
		strResult := string(byteResult)

		if strResult == "[]" {
			exists = true
			return
		}
	}

	errorJson := json.Unmarshal(reply.([]byte), value)

	if errorJson != nil {

		if reflect.TypeOf(value).Kind() == reflect.Ptr && reflect.TypeOf(value).Elem().Kind() == reflect.String {
			var strValue string
			strValue = string(reply.([]byte))

			v := value.(*string)

			*v = strValue

			value = v

			exists = true
			return
		}
		log.Errorf("get %s command result failed:%s", cmd, errorJson.Error())

		err = terror.New(pconst.ERROR_REDIS_GET_UNMARSHAL)

		return
	}

	exists = true

	return
}

// doMGet
func (p *Redis) doMGet(ctx context.Context, cmd string, args []interface{}, value interface{}) (err error) {

	refValue := reflect.ValueOf(value)
	if refValue.Kind() != reflect.Ptr || refValue.Elem().Kind() != reflect.Slice || refValue.Elem().Type().Elem().Kind() != reflect.Ptr {
		log.Errorf("value is not *[]*object:  %v", refValue.Elem().Type().Elem().Kind())
		err = terror.New(pconst.ERROR_REDIS_MGET_TYPE)
		return
	}
	//return errors.New(fmt.Sprintf("s:  %v", refValue.Elem().Type().Elem().Elem()))

	refSlice := refValue.Elem()
	refItem := refSlice.Type().Elem()

	result, errDo := redis.ByteSlices(p.Do(ctx, cmd, args...))

	if errDo != nil {
		log.Errorf("run redis %s command failed: error:%s,args:%v", cmd, errDo.Error(), args)

		err = terror.New(pconst.ERROR_REDIS_MGET_DO)

		return
	}

	if result == nil {
		return nil
	}
	if len(result) > 0 {

		for i := 0; i < len(result); i++ {
			r := result[i]

			if r != nil {
				item := reflect.New(refItem)

				errorJson := json.Unmarshal(r, item.Interface())

				if errorJson != nil {

					log.Errorf("%s command result failed:%s", cmd, errorJson.Error())
					err = terror.New(pconst.ERROR_REDIS_MGET_DO)

					return
				}
				refSlice.Set(reflect.Append(refSlice, item.Elem()))
			} else {
				refSlice.Set(reflect.Append(refSlice, reflect.Zero(refItem)))
			}
		}
	}
	return
}

// doIncr
func (p *Redis) doIncr(ctx context.Context, cmd string, key string, value int64, expire int, fields ...string) (count int64, err error) {

	key = p.getKey(key)

	var data interface{}

	if len(fields) == 0 {
		data, err = p.Do(ctx, cmd, key, value)
	} else {
		field := fields[0]
		data, err = p.Do(ctx, cmd, key, field, value)
	}

	if err != nil {

		return
	}

	count, ok := data.(int64)

	if !ok {

		log.Errorf("get %s command result failed:%v ,is %v", cmd, data, reflect.TypeOf(data))

		err = terror.New(pconst.ERROR_REDIS_INCR_CONVERT)

		return
	}

	if expire == 0 {
		cacheConfig := config.RedisGetBase(p.Persistent)

		expire = cacheConfig.Expire
	}
	//set expire
	if expire > 0 {
		p.Do(ctx, "EXPIRE", key, expire)
	}

	return
}

// doDel
func (p *Redis) doDel(ctx context.Context, cmd string, data ...interface{}) (err error) {

	_, err = p.Do(ctx, cmd, data...)

	return
}

//ZipkinTag
func (p *Redis) ZipkinTag(span opentracing.Span, tag string, err error) {
	if span != nil {
		ext.Error.Set(span, true)
		span.SetTag(tag, err)
	}
}

/**/

// Set
func (p *Redis) Set(ctx context.Context, key string, value interface{}) (err error) {

	_, err = p.doSet(ctx, "SET", key, value, 0)

	return
}

// SetNX
func (p *Redis) SetNX(ctx context.Context, key string, value interface{}) (exists bool, err error) {

	return p.doSetNX(ctx, "SETNX", key, value, 0)
}

// SetEx
func (p *Redis) SetEx(ctx context.Context, key string, value interface{}, expire int) (err error) {

	_, err = p.doSet(ctx, "SET", key, value, expire)

	return
}

// MSet
func (p *Redis) MSet(ctx context.Context, datas map[string]interface{}) (err error) {
	var reply interface{}
	reply, err = p.doMSet(ctx, "MSET", "", datas)

	if err != nil {

		return
	}

	row, ok := reply.(string)

	if !ok || row != "OK" {
		fmt.Println(reply)
		log.Error("MSet reply is not ok,key")
		err = terror.New(pconst.ERROR_REDIS_MSET_REPLY)
	}

	return
}

// Expire
func (p *Redis) Expire(ctx context.Context, key string, expire int) (err error) {
	span, ctx := p.ZipkinNewSpan(ctx, "expire")
	if span != nil {
		defer span.Finish()
	}

	redisResource, err := p.GetConn(ctx)

	if err != nil {
		return
	}

	defer p.PutConn(ctx, redisResource)

	key = p.getKey(key)

	redisClient := redisResource.(ResourceConn)
	_, errDo := redisClient.Do("EXPIRE", key, expire)
	if errDo != nil {
		log.Errorf("run redis EXPIRE command failed: error:%s,key:%s,time:%d", err.Error(), key, expire)

		err = terror.New(pconst.ERROR_REDIS_EXPIRE_DO)

		p.ZipkinTag(span, "do", err)

	}
	return
}

// Get
func (p *Redis) Get(ctx context.Context, key string, data interface{}) (exists bool, err error) {
	return p.doGet(ctx, "GET", key, data)
}

// MGet
func (p *Redis) MGet(ctx context.Context, keys []string, data interface{}) error {

	var args []interface{}

	for _, v := range keys {
		args = append(args, p.getKey(v))
	}

	err := p.doMGet(ctx, "MGET", args, data)

	return err
}

// Incr
func (p *Redis) Incr(ctx context.Context, key string) (count int64, err error) {

	count, err = p.doIncr(ctx, "INCRBY", key, 1, 0)
	return
}

// IncrBy
func (p *Redis) IncrBy(ctx context.Context, key string, value int64) (count int64, err error) {

	count, err = p.doIncr(ctx, "INCRBY", key, value, 0)

	return
}

// Del
func (p *Redis) Del(ctx context.Context, key string) (err error) {

	key = p.getKey(key)

	err = p.doDel(ctx, "DEL", key)

	return
}

// MDel
func (p *Redis) MDel(ctx context.Context, key ...string) (err error) {
	var keys []interface{}
	for _, v := range key {
		keys = append(keys, p.getKey(v))
	}

	err = p.doDel(ctx, "DEL", keys...)

	return
}

/*hash start */

// HIncrby
func (p *Redis) HIncrby(ctx context.Context, key string, field string, value int64) (count int64, err error) {

	return p.doIncr(ctx, "HINCRBY", key, value, 0, field)
}

// HGET
func (p *Redis) HGet(ctx context.Context, key string, field string, value interface{}) (exists bool, err error) {

	exists, err = p.doGet(ctx, "HGET", key, value, field)

	return
}

// HMGet
func (p *Redis) HMGet(ctx context.Context, key string, fields []interface{}, data interface{}) (err error) {
	var args []interface{}

	args = append(args, p.getKey(key))

	for _, v := range fields {
		args = append(args, v)
	}

	err = p.doMGet(ctx, "HMGET", args, data)

	return
}

// HSet
func (p *Redis) HSet(ctx context.Context, key string, field string, value interface{}) (err error) {

	_, err = p.doSet(ctx, "SET", key, value, 0)

	return

}

// HSetNX exists是否存在
func (p *Redis) HSetNX(ctx context.Context, key string, field string, value interface{}) (exists bool, err error) {

	return p.doSetNX(ctx, "HSETNX", key, value, 0, field)
}

// HMSet value是filed:data
func (p *Redis) HMSet(ctx context.Context, key string, value map[string]interface{}) (err error) {
	var reply interface{}
	reply, err = p.doMSet(ctx, "HMSet", key, value)

	if err != nil {

		return
	}

	row, ok := reply.(string)

	if !ok || row != "OK" {
		fmt.Println(reply)
		log.Errorf("doMSet reply is not ok,key:%s", key)
		err = terror.New(pconst.ERROR_REDIS_MSET_REPLY)
	}
	return
}

// HLen
func (p *Redis) HLen(ctx context.Context, key string, data *int) (err error) {

	key = p.getKey(key)
	reply, err := p.Do(ctx, "HLEN", key)

	if err != nil {
		return
	}

	length, b := reply.(int64)

	if !b {
		log.Errorf("redis data convert to int64 failed:%v", reply)
		err = terror.New(pconst.ERROR_REDIS_CONVERT)
		return
	}
	*data = int(length)

	return
}

// HDel
func (p *Redis) HDel(ctx context.Context, key string, data ...interface{}) (err error) {
	var args []interface{}

	key = p.getKey(key)
	args = append(args, key)

	for _, item := range data {
		args = append(args, item)
	}

	err = p.doDel(ctx, "HDEL", args...)

	return
}

// hash end

// ZAdd sorted set
func (p *Redis) ZAdd(ctx context.Context, key string, score int, data interface{}) (err error) {

	key = p.getKey(key)
	_, err = p.Do(ctx, "ZADD", key, score, data)

	return
}

// ZAddM sorted set
func (p *Redis) ZAddM(ctx context.Context, key string, value map[string]interface{}) (err error) {
	_, err = p.doMSet(ctx, "ZADD", key, value)

	return
}

// ZGet
func (p *Redis) ZGet(ctx context.Context, key string, sort bool, start int, end int, value interface{}) (err error) {

	var cmd string
	if sort {
		cmd = "ZRANGE"
	} else {
		cmd = "ZREVRANGE"
	}

	var args []interface{}
	args = append(args, p.getKey(key))
	args = append(args, start)
	args = append(args, end)

	err = p.doMGet(ctx, cmd, args, value)

	return
}

// ZRevRange
func (p *Redis) ZRevRange(ctx context.Context, key string, start int, end int, value interface{}) (err error) {
	return p.ZGet(ctx, key, false, start, end, value)
}

// ZRem
func (p *Redis) ZRem(ctx context.Context, key string, data ...interface{}) (err error) {

	var args []interface{}

	key = p.getKey(key)
	args = append(args, key)

	for _, item := range data {
		args = append(args, item)
	}

	err = p.doDel(ctx, "ZREM", args...)

	return
}

// LRange
func (p *Redis) LRange(ctx context.Context, key string, start int, end int, value interface{}) (err error) {

	cmd := "LRANGE"

	strStart := strconv.Itoa(start)
	strEnd := strconv.Itoa(end)

	_, err = p.doGet(ctx, cmd, key, value, strStart, strEnd)

	return
}

// LLen
func (p *Redis) LLen(ctx context.Context, key string) (len int64, err error) {
	cmd := "LLEN"

	key = p.getKey(key)

	var args []interface{}
	args = append(args, key)
	reply, err := p.Do(ctx, cmd, key)

	if err != nil || reply == nil {
		return
	}

	len, ok := reply.(int64)
	if !ok {
		log.Errorf("LLen reply to int failed,key:%s", key)
		err = terror.New(pconst.ERROR_REDIS_CONVERT)
	}

	return
}

// LRem
func (p *Redis) LRem(ctx context.Context, key string, count int, data interface{}) (r int64, err error) {

	result, err := p.Do(ctx, "LREM", key, count, data)

	if err != nil {
		return
	}

	r, ok := result.(int64)

	if !ok {
		log.Errorf("redis data convert to int failed:%v", result)
		err = terror.New(pconst.ERROR_REDIS_CONVERT)
		return
	}

	return
}

// RPush
func (p *Redis) RPush(ctx context.Context, value interface{}) (err error) {
	cmd := "RPUSH"
	key := ""

	_, err = p.doSet(ctx, cmd, key, value, -1)

	return
}

// LPush
func (p *Redis) LPush(ctx context.Context, value interface{}) (err error) {

	cmd := "LPUSH"
	key := ""

	_, err = p.doSet(ctx, cmd, key, value, -1)

	return
}

// RPop
func (p *Redis) RPop(ctx context.Context, value interface{}) (err error) {
	cmd := "RPOP"

	key := ""

	_, err = p.doGet(ctx, cmd, key, value)

	return
}

// LPop
func (p *Redis) LPop(ctx context.Context, value interface{}) (err error) {

	cmd := "LPOP"

	key := ""

	_, err = p.doGet(ctx, cmd, key, value)

	return
}

//list end

//pipeline start

//PipelineHGet
func (p *Redis) PipelineHGet(ctx context.Context, key []string, fields []interface{}, data []interface{}) (err error) {
	var args [][]interface{}

	for k, v := range key {
		var arg []interface{}
		arg = append(arg, p.getKey(v))
		arg = append(arg, fields[k])
		args = append(args, arg)
	}

	err = p.PipeDo(ctx, "HGET", args, data)

	return
}

//pipeline end

// SAdd Set集合Start
func (p *Redis) SAdd(ctx context.Context, key string, argPs []interface{}) (err error) {

	args := make([]interface{}, len(argPs)+1)
	args[0] = p.getKey(key)
	copy(args[1:], argPs)

	_, err = p.Do(ctx, "SADD", args...)

	return
}

// SIsMember
func (p *Redis) SIsMember(ctx context.Context, key string, arg interface{}) (err error) {

	key = p.getKey(key)

	reply, err := p.Do(ctx, "SISMEMBER", key, arg)

	if err != nil {

		return
	}
	if code, ok := reply.(int64); !ok || code != int64(1) {
		err = terror.New(pconst.ERROR_REDIS_CONVERT)
	}
	return
}

// SRem
func (p *Redis) SRem(ctx context.Context, key string, argPs []interface{}) (err error) {

	args := make([]interface{}, len(argPs)+1)
	args[0] = p.getKey(key)
	copy(args[1:], argPs)

	_, err = p.Do(ctx, "SREM", args...)

	if err != nil {
		return
	}
	return
}

// HGetAll 不建议使用
func (p *Redis) HGetAll(ctx context.Context, key string, data interface{}) (err error) {
	var args []interface{}

	args = append(args, p.getKey(key))

	err = p.doMGet(ctx, "HGETALL", args, data)

	return
}
