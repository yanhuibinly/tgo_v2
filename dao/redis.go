package dao

import (
	"github.com/youtube/vitess/go/pools"
	"github.com/garyburd/redigo/redis"
	"github.com/tonyjt/tgo_v2/config"
	"github.com/tonyjt/tgo_v2/log"
	"github.com/tonyjt/tgo_v2/terror"
	"github.com/tonyjt/tgo_v2/pconst"
	"time"
	"strings"
	"reflect"
	"fmt"
	"context"
	"github.com/opentracing/opentracing-go"
	"encoding/json"
)


var(
	unpersist     *pools.ResourcePool
	persist    *pools.ResourcePool // 持久化Pool
)


type ResourceConn struct {
	redis.Conn
	serverIndex int
}

func init(){
	if config.FeatureRedis(){
		//非持久化pool
		conf:= config.RedisGet()
		unpersist = initRedisPool(conf.Unpersist)
		persist = initRedisPool(conf.Persist)
	}
}

func initRedisPool(conf config.RedisBase)*pools.ResourcePool{
	fmt.Println(conf)
	return pools.NewResourcePool(func() (pools.Resource, error) {
		c, serverIndex, err := dial(0,conf)
		return ResourceConn{Conn: c, serverIndex: serverIndex}, err
	}, conf.PoolMinActive, conf.PoolMaxActive, time.Duration(conf.PoolIdleTimeout)*time.Millisecond)

}
func dial(fromIndex int,config config.RedisBase) (conn redis.Conn, index int, err error) {

	fmt.Println(config.Address)
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

func (r ResourceConn) Close() {
	r.Conn.Close()
}



type Redis struct{
	Key string
	Persistent bool
}

func (p *Redis) ZipkinNewSpan(ctx context.Context,name string)(opentracing.Span,context.Context){
	if config.FeatureZipkin(){
		return opentracing.StartSpanFromContext(ctx,fmt.Sprintf("redis:%s",name))
	}else{
		return nil,ctx
	}
}

func (p *Redis) GetConn(ctx context.Context)(conn pools.Resource,err error){

	span, ctx := p.ZipkinNewSpan(ctx, "conn")
	if span != nil {
		defer span.Finish()
	}

	var pool *pools.ResourcePool

	if p.Persistent{
		pool = persist
	}else{
		pool = unpersist
	}

	if pool ==nil{
		log.Logf(log.LevelFatal,"redis pool is null")
		err =terror.New(pconst.ERROR_REDIS_POOL_NULL)
		if span !=nil{
			span.SetTag("err:pool",err)
		}
		return
	}

	r,err :=pool.Get(ctx)

	if err!=nil{
		log.Errorf("redis get connection err:%s", err.Error())
		err =terror.New(pconst.ERROR_REDIS_POOL_GET)
		if span !=nil{
			span.SetTag("err:pool",err)
		}
		return
	}

	if r == nil{
		log.Error("redis pool resource is null")
		err = terror.New(pconst.ERROR_REDIS_POOL_EMPTY)
		if span !=nil{
			span.SetTag("err:pool",err)
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

		if p.Persistent{
			conf = config.RedisGet().Persist
		}else{
			conf = config.RedisGet().Unpersist
		}
		c, serverIndex, err = dial(rc.serverIndex + 1,conf)
		if err != nil {
			pool.Put(r)
			log.Errorf("redis redail connection err:%s", err.Error())
			err = terror.New(pconst.ERROR_REDIS_POOL_REDIAL)
			if span !=nil{
				span.SetTag("err:dial",err)
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

func (p *Redis) PutConn(ctx context.Context,resource pools.Resource){
	var pool *pools.ResourcePool

	if p.Persistent{
		pool = persist
	}else{
		pool = unpersist
	}

	pool.Put(resource)
}
func (p *Redis) getKey(key string) string {

	conf:= config.RedisGetBase(p.Persistent)

	prefixRedis := conf.Prefix

	if strings.Trim(key, " ") == "" {
		return fmt.Sprintf("%s:%s", prefixRedis, p.Key)
	}
	return fmt.Sprintf("%s:%s:%s", prefixRedis, p.Key, key)
}
// doSet
func (p *Redis) doSet(ctx context.Context,cmd string, key string, value interface{}, expire int, fields ...string) (reply interface{},err error) {

	span, ctx := p.ZipkinNewSpan(ctx, "doset")
	if span != nil {
		defer span.Finish()
	}

	redisResource, err:= p.GetConn(ctx)

	if err != nil {
		return
	}

	key = p.getKey(key)

	defer p.PutConn(ctx,redisResource)

	redisClient := redisResource.(ResourceConn)

	data, errJson := json.Marshal(value)

	if errJson != nil {

		log.Errorf("redis %s marshal data to json:%s", cmd, errJson.Error())

		err = terror.New(pconst.ERROR_REDIS_SET_MARSHAL)
		p.ZipkinTag(span,"marshal",err)

		return
	}

	if expire == 0 {
		cacheConfig := config.RedisGetBase(p.Persistent)

		expire = cacheConfig.Expire
	}

	var errDo error

	if len(fields) == 0 {
		if expire > 0 && strings.ToUpper(cmd) == "SET" {
			reply, errDo = redisClient.Do(cmd, key, data, "ex", expire)
		} else {
			reply, errDo = redisClient.Do(cmd, key, data)
		}

	} else {
		field := fields[0]

		reply, errDo = redisClient.Do(cmd, key, field, data)

	}

	if errDo != nil {
		log.Errorf("run redis command %s failed:error:%s,key:%s,fields:%v,data:%v", cmd, errDo.Error(), key, fields, value)

		err = terror.New(pconst.ERROR_REDIS_SET_DO)
		p.ZipkinTag(span,"do",err)
		return
	}
	//set expire
	if expire > 0 && strings.ToUpper(cmd) != "SET" {
		_, errExpire := redisClient.Do("EXPIRE", key, expire)
		if errExpire != nil {
			log.Errorf("run redis EXPIRE command failed: error:%s,key:%s,time:%d", errExpire.Error(), key, expire)
			p.ZipkinTag(span,"expire",terror.NewFromError(errExpire))
		}
	}

	return
}

// doSetNX
func (p *Redis) doSetNX(ctx context.Context,cmd string, key string, value interface{}, expire int, field ...string) (row int64, err error) {

	reply, err := p.doSet(ctx,cmd, key, value, expire, field...)

	if err != nil {
		return
	}

	row, ok := reply.(int64)

	if !ok {
		log.Errorf("HSetNX reply to int failed,key:%s,field:%s", key, field)
		err = terror.New(pconst.ERROR_REDIS_SETNX_REPLY)
		return
	}

	return
}

// doMSet
func (p *Redis) doMSet(ctx context.Context,cmd string, key string, value map[string]interface{}) (reply interface{}, err error) {

	span, ctx := p.ZipkinNewSpan(ctx, "domset")
	if span != nil {
		defer span.Finish()
	}

	redisResource, err:= p.GetConn(ctx)

	if err != nil {
		return
	}

	defer p.PutConn(ctx,redisResource)


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
			p.ZipkinTag(span,"marshal",err)
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

	redisClient := redisResource.(ResourceConn)

	var errDo error
	reply, errDo = redisClient.Do(cmd, args...)

	if errDo != nil {
		log.Errorf("run redis command %s failed:error:%s,key:%s,value:%v", cmd, errDo.Error(), key, value)

		err = terror.New(pconst.ERROR_REDIS_MSET_DO)
		p.ZipkinTag(span,"do",err)
		return
	}
	return
}

// doGet
func (p *Redis) doGet(ctx context.Context,cmd string, key string, value interface{}, fields ...string) (exists bool,err error) {


	span, ctx := p.ZipkinNewSpan(ctx, "doget")
	if span != nil {
		defer span.Finish()
	}

	redisResource, err:= p.GetConn(ctx)

	if err != nil {
		return
	}

	defer p.PutConn(ctx,redisResource)


	redisClient := redisResource.(ResourceConn)

	key = p.getKey(key)

	var result interface{}
	var errDo error

	//if len(fields) == 0 {
	//	result, errDo = redisClient.Do(cmd, key)
	//} else {
	var args []interface{}

	args = append(args, key)

	for _, f := range fields {
		args = append(args, f)
	}

	result, errDo = redisClient.Do(cmd, args...)

	if errDo != nil {

		log.Errorf("run redis %s command failed: error:%s,key:%s,fields:%v", cmd, errDo.Error(), key, fields)
		err = terror.New(pconst.ERROR_REDIS_GET_DO)
		p.ZipkinTag(span,"do",err)
		return
	}

	if result == nil {
		value = nil
		exists = false
		return
	}

	if reflect.TypeOf(result).Kind() == reflect.Slice {

		byteResult := (result.([]byte))
		strResult := string(byteResult)

		if strResult == "[]" {
			exists = true
			return
		}
	}

	errorJson := json.Unmarshal(result.([]byte), value)

	if errorJson != nil {

		if reflect.TypeOf(value).Kind() == reflect.Ptr && reflect.TypeOf(value).Elem().Kind() == reflect.String {
			var strValue string
			strValue = string(result.([]byte))

			v := value.(*string)

			*v = strValue

			value = v

			exists = true
			return
		}
		log.Errorf("get %s command result failed:%s", cmd, errorJson.Error())

		err = terror.New(pconst.ERROR_REDIS_GET_UNMARSHAL)
		p.ZipkinTag(span,"unmarshal",err)

		return
	}

	exists = true

	return
}

// doMGet
func (p *Redis) doMGet(ctx context.Context,cmd string, args []interface{}, value interface{}) (err error) {

	span, ctx := p.ZipkinNewSpan(ctx, "doMGet")
	if span != nil {
		defer span.Finish()
	}


	refValue := reflect.ValueOf(value)
	if refValue.Kind() != reflect.Ptr || refValue.Elem().Kind() != reflect.Slice || refValue.Elem().Type().Elem().Kind() != reflect.Ptr {
		log.Errorf("value is not *[]*object:  %v", refValue.Elem().Type().Elem().Kind())
		err = terror.New(pconst.ERROR_REDIS_MGET_TYPE)
		p.ZipkinTag(span,"replay",err)
		return
	}
	//return errors.New(fmt.Sprintf("s:  %v", refValue.Elem().Type().Elem().Elem()))

	refSlice := refValue.Elem()
	refItem := refSlice.Type().Elem()


	redisResource, err:= p.GetConn(ctx)

	if err != nil {
		return
	}

	defer p.PutConn(ctx,redisResource)

	redisClient := redisResource.(ResourceConn)

	result, errDo := redis.ByteSlices(redisClient.Do(cmd, args...))

	if errDo != nil {
		log.Errorf("run redis %s command failed: error:%s,args:%v", cmd, errDo.Error(), args)

		err = terror.New(pconst.ERROR_REDIS_MGET_DO)

		p.ZipkinTag(span,"do",err)

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

					p.ZipkinTag(span,"do",err)

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
func (p *Redis) doIncr(ctx context.Context,cmd string, key string, value int, expire int, fields ...string) (count int64,err error) {

	span, ctx := p.ZipkinNewSpan(ctx, "doincr")
	if span != nil {
		defer span.Finish()
	}

	redisResource, err:= p.GetConn(ctx)

	if err != nil {
		return
	}

	defer p.PutConn(ctx,redisResource)


	redisClient := redisResource.(ResourceConn)

	key = p.getKey(key)

	var data interface{}
	var errDo error

	if len(fields) == 0 {
		data, errDo = redisClient.Do(cmd, key, value)
	} else {
		field := fields[0]
		data, errDo = redisClient.Do(cmd, key, field, value)
	}

	if errDo != nil {
		log.Errorf("run redis %s command failed: error:%s,key:%s,fields:%v,value:%d", cmd, errDo.Error(), key, fields, value)

		err = terror.New(pconst.ERROR_REDIS_SET_MARSHAL)

		p.ZipkinTag(span,"do",err)
		return
	}

	count, ok := data.(int64)

	if !ok {

		log.Errorf("get %s command result failed:%v ,is %v", cmd, data, reflect.TypeOf(data))

		err = terror.New(pconst.ERROR_REDIS_INCR_CONVERT)

		p.ZipkinTag(span,"do",err)
		return
	}

	if expire == 0 {
		cacheConfig := config.RedisGetBase(p.Persistent)

		expire = cacheConfig.Expire
	}
	//set expire
	if expire > 0 {
		_, errExpire := redisClient.Do("EXPIRE", key, expire)
		if errExpire != nil {
			log.Errorf("run redis EXPIRE command failed: error:%s,key:%s,time:%d", errExpire.Error(), key, expire)
			p.ZipkinTag(span,"expire",terror.NewFromError(errExpire))
		}
	}

	return
}
// doDel
func (p *Redis) doDel(ctx context.Context,cmd string, data ...interface{}) (err error) {

	span, ctx := p.ZipkinNewSpan(ctx, "dodel")
	if span != nil {
		defer span.Finish()
	}

	redisResource, err:= p.GetConn(ctx)

	if err != nil {
		return
	}

	defer p.PutConn(ctx,redisResource)

	redisClient := redisResource.(ResourceConn)

	_, errDo := redisClient.Do(cmd, data...)

	if errDo != nil {
		log.Errorf("run redis %s command failed: error:%s,data:%v", cmd, errDo.Error(), data)

		err = terror.New(pconst.ERROR_REDIS_DEL_DO)

		p.ZipkinTag(span, "do", err)
	}

	return
}


func (p *Redis) ZipkinTag(span opentracing.Span,tag string,err error){
	if span !=nil{
		span.SetTag(tag,err)
	}
}
/**/

// Set
func (p *Redis) Set(ctx context.Context,key string, value interface{}) (err error) {

	_,err = p.doSet(ctx,"SET", key, value, 0)

	return
}
// SetNX
func (p *Redis) SetNX(ctx context.Context,key string, value interface{}) (int64, error) {

	return p.doSetNX(ctx,"SETNX", key, value, 0)
}

// SetEx
func (p *Redis) SetEx(ctx context.Context,key string, value interface{}, expire int) (err error) {

	_, err = p.doSet(ctx,"SET", key, value, expire)

	return
}

// MSet
func (p *Redis) MSet(ctx context.Context,datas map[string]interface{}) (err error) {
	_, err = p.doMSet(ctx,"MSET", "", datas)

	return
}

// Expire
func (p *Redis) Expire(ctx context.Context,key string,expire int)(err error){
	span, ctx := p.ZipkinNewSpan(ctx, "expire")
	if span != nil {
		defer span.Finish()
	}

	redisResource, err:= p.GetConn(ctx)

	if err != nil {
		return
	}

	defer p.PutConn(ctx,redisResource)

	key = p.getKey(key)

	redisClient := redisResource.(ResourceConn)
	_, errDo := redisClient.Do("EXPIRE", key, expire)
	if errDo != nil {
		log.Errorf("run redis EXPIRE command failed: error:%s,key:%s,time:%d", err.Error(), key, expire)

		err = terror.New(pconst.ERROR_REDIS_EXPIRE_DO)

		p.ZipkinTag(span,"do",err)

	}
	return
}

// Get
func (p *Redis) Get(ctx context.Context,key string,data interface{})(exists bool,err error){
	return p.doGet(ctx,"GET", key, data)
}

// MGet
func (p *Redis) MGet(ctx context.Context,keys []string, data interface{}) error {

	var args []interface{}

	for _, v := range keys {
		args = append(args, p.getKey(v))
	}

	err := p.doMGet(ctx,"MGET", args, data)

	return err
}

// Incr
func (p *Redis) Incr(ctx context.Context,key string) (count int64,err error) {

	count,err =  p.doIncr(ctx,"INCRBY", key, 1, 0)
	return
}
// IncrBy
func (p *Redis) IncrBy(ctx context.Context,key string, value int) (count int64, err error) {

	count,err = p.doIncr(ctx,"INCRBY", key, value, 0)
	return
}

// Del
func (p *Redis) Del(ctx context.Context,key string) (err error) {

	key = p.getKey(key)

	err = p.doDel(ctx,"DEL", key)

	return
}

// MDel
func (p *Redis) MDel(ctx context.Context,key ...string) (err error) {
	var keys []interface{}
	for _, v := range key {
		keys = append(keys, p.getKey(v))
	}

	err = p.doDel(ctx,"DEL", keys...)

	return
}

/*hash start */

// HIncrby
func (p *Redis) HIncrby(ctx context.Context,key string, field string, value int) (count int64, err error) {

	return p.doIncr(ctx,"HINCRBY", key, value, 0, field)
}

// HGET
func (p *Redis) HGet(ctx context.Context,key string, field string, value interface{}) (exists bool,err error) {

	exists, err = p.doGet(ctx,"HGET", key, value, field)

	return
}

// HMGet
func (p *Redis) HMGet(ctx context.Context,key string, fields []interface{}, data interface{}) (err error) {
	var args []interface{}

	args = append(args, p.getKey(key))

	for _, v := range fields {
		args = append(args, v)
	}

	err = p.doMGet(ctx,"HMGET", args, data)

	return
}

// HSet
func (p *Redis) HSet(ctx context.Context,key string, field string, value interface{}) (err error) {

	_,err = p.doSet(ctx,"SET", key, value, 0)

	return

}
/*
// HSetNX
func (p *Redis) HSetNX(ctx context.Context,key string, field string, value interface{}) (int64, err error) {

	return p.doSetNX(ctx,"HSETNX", key, value, 0, field)
}

//HMSet value是filed:data
func (p *Redis) HMSet(key string, value map[string]interface{}) bool {
	_, err := p.doMSet("HMSet", key, value)
	if err != nil {
		return false
	}
	return true
}

//HMSetE value是filed:data
func (p *Redis) HMSetE(key string, value map[string]interface{}) error {
	_, err := p.doMSet("HMSet", key, value)
	return err
}

func (p *Redis) HLen(key string, data *int) bool {
	redisResource, err := p.InitRedisPool()

	if err != nil {
		return false
	}
	defer daoPool.Put(redisResource, p.Persistent)

	redisClient := redisResource.(ResourceConn)
	key = p.getKey(key)
	resultData, errDo := redisClient.Do("HLEN", key)

	if errDo != nil {
		log.Errorf("run redis HLEN command failed: error:%s,key:%s", errDo.Error(), key)
		return false
	}

	length, b := resultData.(int64)

	if !b {
		log.Errorf("redis data convert to int64 failed:%v", resultData)
	}
	*data = int(length)

	return b
}

func (p *Redis) HDel(key string, data ...interface{}) bool {
	var args []interface{}

	key = p.getKey(key)
	args = append(args, key)

	for _, item := range data {
		args = append(args, item)
	}

	err := p.doDel("HDEL", args...)

	if err != nil {
		log.Errorf("run redis HDEL command failed: error:%s,key:%s,data:%v", err.Error(), key, data)
		return false
	}

	return true
}

// hash end

// sorted set start
func (p *Redis) ZAdd(key string, score int, data interface{}) bool {
	redisResource, err := p.InitRedisPool()

	if err != nil {
		return false
	}
	defer daoPool.Put(redisResource, p.Persistent)

	redisClient := redisResource.(ResourceConn)
	key = p.getKey(key)
	_, errDo := redisClient.Do("ZADD", key, score, data)

	if errDo != nil {
		log.Errorf("run redis ZADD command failed: error:%s,key:%s,score:%d,data:%v", errDo.Error(), key, score, data)
		return false
	}
	return true
}

// sorted set start
func (p *Redis) ZAddM(key string, value map[string]interface{}) bool {
	_, err := p.doMSet("ZADD", key, value)
	if err != nil {
		return false
	}
	return true
}

func (p *Redis) ZGet(key string, sort bool, start int, end int, value interface{}) error {

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

	err := p.doMGet(cmd, args, value)

	return err
}

func (p *Redis) ZRevRange(key string, start int, end int, value interface{}) error {
	return p.ZGet(key, false, start, end, value)
}

func (p *Redis) ZRem(key string, data ...interface{}) bool {

	var args []interface{}

	key = p.getKey(key)
	args = append(args, key)

	for _, item := range data {
		args = append(args, item)
	}

	err := p.doDel("ZREM", args...)

	if err != nil {
		return false
	}
	return true
}

//list start

func (p *Redis) LRange(key string, start int, end int, value interface{}) bool {

	cmd := "LRANGE"

	strStart := strconv.Itoa(start)
	strEnd := strconv.Itoa(end)
	_, err := p.doGet(cmd, key, value, strStart, strEnd)
	if err == nil {
		return true
	} else {
		return false
	}
}

func (p *Redis) LLen(key string) (int64, error) {
	cmd := "LLEN"

	redisResource, err := p.InitRedisPool()

	if err != nil {
		return 0, err
	}
	defer daoPool.Put(redisResource, p.Persistent)

	redisClient := redisResource.(ResourceConn)
	key = p.getKey(key)

	var result interface{}
	var errDo error

	var args []interface{}
	args = append(args, key)
	result, errDo = redisClient.Do(cmd, key)

	if errDo != nil {

		log.Errorf("run redis %s command failed: error:%s,key:%s", cmd, errDo.Error(), key)

		return 0, errDo
	}

	if result == nil {
		return 0, nil
	}

	num, ok := result.(int64)
	if !ok {
		return 0, errors.New("result to int64 failed")
	}

	return num, nil
}

func (p *Redis) LREM(key string, count int, data interface{}) int {
	redisResource, err := p.InitRedisPool()

	if err != nil {
		return 0
	}
	defer daoPool.Put(redisResource, p.Persistent)

	redisClient := redisResource.(ResourceConn)

	result, errDo := redisClient.Do("LREM", key, count, data)

	if errDo != nil {
		log.Errorf("run redis command LREM failed: error:%s,key:%s,count:%d,data:%v", errDo.Error(), key, count, data)
		return 0
	}

	countRem, ok := result.(int)

	if !ok {
		log.Errorf("redis data convert to int failed:%v", result)
		return 0
	}

	return countRem
}

func (p *Redis) RPush(value interface{}) bool {
	return p.Push(value, false)
}
func (p *Redis) LPush(value interface{}) bool {
	return p.Push(value, true)
}

func (p *Redis) Push(value interface{}, isLeft bool) bool {

	var cmd string
	if isLeft {
		cmd = "LPUSH"
	} else {
		cmd = "RPUSH"
	}

	key := ""

	_, err := p.doSet(cmd, key, value, -1)

	if err != nil {
		return false
	}
	return true
}

func (p *Redis) RPop(value interface{}) bool {
	return p.Pop(value, false)
}

func (p *Redis) LPop(value interface{}) bool {
	return p.Pop(value, true)
}

func (p *Redis) Pop(value interface{}, isLeft bool) bool {

	var cmd string
	if isLeft {
		cmd = "LPOP"
	} else {
		cmd = "RPOP"
	}
	key := ""

	_, err := p.doGet(cmd, key, value)

	if err == nil {
		return true
	} else {
		return false
	}
}

//list end

//pipeline start

func (p *Redis) PipelineHGet(key []string, fields []interface{}, data []interface{}) error {
	var args [][]interface{}

	for k, v := range key {
		var arg []interface{}
		arg = append(arg, p.getKey(v))
		arg = append(arg, fields[k])
		args = append(args, arg)
	}

	err := p.pipeDoGet("HGET", args, data)

	return err
}

func (p *Redis) pipeDoGet(cmd string, args [][]interface{}, value []interface{}) error {

	redisResource, err := p.InitRedisPool()

	if err != nil {
		return err
	}
	defer daoPool.Put(redisResource, p.Persistent)

	redisClient := redisResource.(ResourceConn)

	for _, v := range args {
		if err := redisClient.Send(cmd, v...); err != nil {
			log.Errorf("Send(%v) returned error %v", v, err)
			return err
		}
	}
	if err := redisClient.Flush(); err != nil {
		log.Errorf("Flush() returned error %v", err)
		return err
	}
	for k, v := range args {
		result, err := redisClient.Receive()
		if err != nil {
			log.Errorf("Receive(%v) returned error %v", v, err)
			return err
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
			return errorJson
		}
	}

	return nil
}

//pipeline end

// Set集合Start
func (p *Redis) SAdd(key string, argPs []interface{}) bool {
	redisResource, err := p.InitRedisPool()

	if err != nil {
		return false
	}
	defer daoPool.Put(redisResource, p.Persistent)

	args := make([]interface{}, len(argPs)+1)
	args[0] = p.getKey(key)
	copy(args[1:], argPs)

	redisClient := redisResource.(ResourceConn)

	_, errDo := redisClient.Do("SADD", args...)

	if errDo != nil {
		log.Errorf("run redis SADD command failed: error:%s,key:%s,args:%v", errDo.Error(), key, args)
		return false
	}
	return true
}

func (p *Redis) SIsMember(key string, arg interface{}) bool {
	redisResource, err := p.InitRedisPool()

	if err != nil {
		return false
	}
	defer daoPool.Put(redisResource, p.Persistent)

	key = p.getKey(key)

	redisClient := redisResource.(ResourceConn)
	reply, errDo := redisClient.Do("SISMEMBER", key, arg)

	if errDo != nil {
		log.Errorf("run redis SISMEMBER command failed: error:%s,key:%s,member:%s", errDo.Error(), key, arg)
		return false
	}
	if code, ok := reply.(int64); ok && code == int64(1) {
		return true
	}
	return false
}

func (p *Redis) SRem(key string, argPs []interface{}) bool {
	redisResource, err := p.InitRedisPool()

	if err != nil {
		return false
	}
	defer daoPool.Put(redisResource, p.Persistent)

	args := make([]interface{}, len(argPs)+1)
	args[0] = p.getKey(key)
	copy(args[1:], argPs)

	redisClient := redisResource.(ResourceConn)

	_, errDo := redisClient.Do("SREM", args...)

	if errDo != nil {
		log.Errorf("run redis SREM command failed: error:%s,key:%s,member:%s", errDo.Error(), key, args)
		return false
	}
	return true
}

func (p *Redis) HGetAll(key string, data interface{}) error {
	var args []interface{}

	args = append(args, p.getKey(key))

	err := p.doMGet("HGETALL", args, data)

	return err
}
*/