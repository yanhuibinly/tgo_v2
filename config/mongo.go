package config

type Mongo struct {
	Mongo []MongoConf
}
type MongoConf struct {
	Db   string
	Conn MongoConn
}

type MongoConn struct {
	Servers    string
	ReadOption string `json:"read_option"`
	Timeout    int
	PoolLimit  int `json:"pool_limit"`
}

var (
	mongoConfig map[string]MongoConf
)

func init() {
	if FeatureMongo() {
		config := &Mongo{}

		defaultMongoConfig := configMongoGetDefault()

		configGet("mongo", config, defaultMongoConfig)

		if len(config.Mongo) == 0 {
			panic("mongo config is empty")
		}

		mongoConfig = make(map[string]MongoConf)

		for i, c := range config.Mongo {
			mongoConfig[c.Db] = config.Mongo[i]
		}
	}
}

func configMongoGetDefault() *Mongo {
	return &Mongo{Mongo: []MongoConf{MongoConf{Db: "tgo", Conn: MongoConn{
		Servers: "servers", ReadOption: "PRIMARY", Timeout: 1000, PoolLimit: 30}}}}
}

func MongoGet(dbName string) MongoConf {
	return mongoConfig[dbName]
}
func MongoGetAll() map[string]MongoConf {

	return mongoConfig
}
