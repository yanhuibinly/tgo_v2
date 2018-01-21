package config

//Feature feature struct
type Feature struct {
	Zipkin bool
	Mongo  bool
	Mysql  bool
	Redis  bool
	Grpc   bool
	HTTP   bool
	Es     bool
}

var (
	featureConfig *Feature
)

func init() {
	featureConfig = &Feature{}

	err := configGet("feature", featureConfig, false, nil)

	if err != nil {
		defaultFeatureConfig := configFeatureGetDefault()

		featureConfig = defaultFeatureConfig
	}
}

//configFeatureGetDefault get default feature config
func configFeatureGetDefault() *Feature {
	return &Feature{Zipkin: true, Mongo: true, Mysql: true, Redis: true, Grpc: true, HTTP: true}
}

//FeatureGet get feature
func FeatureGet() *Feature {
	if featureConfig == nil {
		panic("feature config is nil")
	}
	return featureConfig
}

//FeatureMongo get mongo feature
func FeatureMongo() bool {
	return featureConfig.Mongo
}

//FeatureMysql get mysql feature
func FeatureMysql() bool {
	return featureConfig.Mysql
}

//FeatureZipkin get zipkin feature
func FeatureZipkin() bool {
	return featureConfig.Zipkin
}

//FeatureRedis get redis feature
func FeatureRedis() bool {
	return featureConfig.Redis
}

//FeatureGrpc get grpc feature
func FeatureGrpc() bool {
	return featureConfig.Grpc
}

//FeatureHTTP get http feature
func FeatureHTTP() bool {
	return featureConfig.HTTP
}

//FeatureEs get es feature
func FeatureEs() bool {
	return featureConfig.Es
}
