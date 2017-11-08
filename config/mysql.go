package config

type Mysql struct{
	DbName string
	Write MysqlBase
	Reads []MysqlBase
	Pool	MysqlPool
}

type MysqlBase struct {
	Address  string
	Port     int
	User     string
	Password string
}

type MysqlPool struct {
	Max int
	IdleMax int
	LifeTimeSeconds int
}

var (
	mysqlConfig *Mysql
)
func init(){
	if FeatureMysql() {
		mysqlConfig = &Mysql{}

		defaultMysqlConfig := configMysqlGetDefault()

		configGet("mysql", mysqlConfig, defaultMysqlConfig)
	}
}



func configMysqlGetDefault() *Mysql {
	return &Mysql{
		DbName: "",
		Write:  MysqlBase{"ip", 33062, "user", "password"},
		Reads: []MysqlBase{MysqlBase{"ip", 3306, "user", "password"}},
		Pool: MysqlPool{Max:16,IdleMax:5,LifeTimeSeconds:0},
	}
}

func MysqlGet() *Mysql {

	return mysqlConfig
}
