package config

type Mongo struct {
	DbName      string
	Servers     string
	ReadOption string `json:"read_option"`
	Timeout     int
	PoolLimit   int `json:"pool_limit"`
}

var (
	mongoConfig *Mongo
)
func init(){
	if FeatureMongo() {
		mongoConfig = &Mongo{}

		defaultMongoConfig := configMongoGetDefault()

		configGet("mongo", mongoConfig, defaultMongoConfig)
	}
}



func configMongoGetDefault() *Mongo {
	return &Mongo{DbName: "dbname", Servers: "servers", ReadOption: "PRIMARY", Timeout: 1000, PoolLimit: 30}
}

func MongoGet() *Mongo {

	return mongoConfig
}

