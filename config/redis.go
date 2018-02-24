package config

var (
	redisConfig *Redis
)

type Redis struct {
	Unpersist RedisBase //非持久化,默认使用
	Persist   RedisBase // 持久化Redis
}

type RedisBase struct {
	Address         []string
	Prefix          string
	Expire          int
	ReadTimeout     int
	WriteTimeout    int
	ConnectTimeout  int
	PoolMaxIdle     int
	PoolMaxActive   int
	PoolIdleTimeout int
	PoolMinActive   int
	Password string
}

func init() {
	if FeatureRedis() {
		redisConfig = &Redis{}

		err := configGet("redis", redisConfig, false, nil)

		if err != nil {
			panic("redis config error")
		}
	}
}

func configRedisGetDefault() *Redis {
	return &Redis{Unpersist: RedisBase{[]string{"ip:port"}, "prefix", 604800, 1000, 1000, 1000, 10, 100, 180000, 2,""},
		Persist: RedisBase{[]string{"ip:port"}, "prefix", 604800, 1000, 1000, 1000, 10, 100, 180000, 2,""}}
}

	func RedisGet() *Redis {
		if redisConfig == nil {
			panic("redis config is nill")
		}
		return redisConfig
	}

func RedisGetBase(persistent bool) RedisBase {

	conf := RedisGet()

	if !persistent {
		return conf.Unpersist
	}
	return conf.Persist
}
