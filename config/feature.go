package config

type Feature struct {
	Zipkin bool
	Mongo bool
	Mysql bool
	Redis bool
}

var (
	featureConfig *Feature
)
func init(){
	featureConfig = &Feature{}

	defaultFeatureConfig := configFeatureGetDefault()

	configGet("feature", featureConfig, defaultFeatureConfig)
}



func configFeatureGetDefault() *Feature {
	return &Feature{Zipkin:true,Mongo:true,Mysql:true,Redis:true}
}

func FeatureGet() *Feature {

	return featureConfig
}

func FeatureMongo() bool{
	return featureConfig.Mongo
}

func FeatureMysql() bool{
	return featureConfig.Mysql
}

func FeatureZipkin() bool{
	return featureConfig.Zipkin
}

func FeatureRedis() bool{
	return featureConfig.Redis
}

