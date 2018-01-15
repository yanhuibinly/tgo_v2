package config

type Es struct {
	Es []EsConf
}

type EsConf struct {
	Service string
	Conn    []string
}

var (
	esConfig map[string]*EsConf
)

func init() {

	if FeatureEs() {
		config := &Es{}

		err := configGet("es", config, false, nil)

		if err != nil || len(config.Es) == 0 {
			panic("es config is empty")
		}

		esConfig = make(map[string]*EsConf)

		for i, c := range config.Es {
			esConfig[c.Service] = &config.Es[i]
		}
	}
}

func configEsGetDefault() *Es {
	return &Es{Es: []EsConf{EsConf{Service: "tgo", Conn: []string{}}}}
}

func EsGet(service string) *EsConf {
	if esConfig == nil {
		panic("es config is nil")
	}
	g, ok := esConfig[service]

	if !ok {
		return nil
	}
	return g
}
