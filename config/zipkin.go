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

		err := configGet("zipkin", zipkinConfig, false, nil)

		if err != nil {
			defaultZipkinConfig := configZipkinGetDefault()

			zipkinConfig = defaultZipkinConfig
		}
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
	if zipkinConfig == nil {
		panic("zipconfig is nil")
	}
	return zipkinConfig
}
