package config

type ConfigZipkin struct {
	ServiceName       string
	CollectorEndpoint string
	Debug             bool
	SameSpan          bool
	TraceID128Bit     bool
}

var (
	zipkinConfig *ConfigZipkin
)

func init() {
	if FeatureZipkin() {
		zipkinConfig = &ConfigZipkin{}

		defaultZipkinConfig := configZipkinGetDefault()

		configGet("zipkin", zipkinConfig, defaultZipkinConfig)
	}
}

func configZipkinGetDefault() *ConfigZipkin {
	return &ConfigZipkin{
		ServiceName:       "tgov2",
		CollectorEndpoint: "172.172.177.19:9411/api/v1/spans",
		Debug:             false,
		SameSpan:          true,
		TraceID128Bit:     true}
}

func ZipkinGet() *ConfigZipkin {

	return zipkinConfig
}
