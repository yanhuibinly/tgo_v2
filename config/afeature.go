package config

type Feature struct {
	Zipkin bool
	Mongo  bool
	Mysql  bool
	Redis  bool
	Grpc   bool
	Http   bool
	Es     bool
}

var (
	featureConfig *Feature
)

func init() {
	featureConfig = &Feature{}

	defaultFeatureConfig := configFeatureGetDefault()

	configGet("feature", featureConfig, defaultFeatureConfig)
}

func configFeatureGetDefault() *Feature {
	return &Feature{Zipkin: true, Mongo: true, Mysql: true, Redis: true, Grpc: true, Http: true}
}

func FeatureGet() *Feature {
	if featureConfig == nil {
		panic("feature config is nil")
	}
	return featureConfig
}

func FeatureMongo() bool {
	return featureConfig.Mongo
}

func FeatureMysql() bool {
	return featureConfig.Mysql
}

func FeatureZipkin() bool {
	return featureConfig.Zipkin
}

func FeatureRedis() bool {
	return featureConfig.Redis
}

func FeatureGrpc() bool {
	return featureConfig.Grpc
}

func FeatureHttp() bool {
	return featureConfig.Http
}

func FeatureEs() bool {
	return featureConfig.Es
}
