package config

type Config struct {
	ZapConfig
	DatabaseConfig
}

type ZapConfig struct {
	Filename   string
	MaxSize    int
	MaxAge     int
	MaxBackups int
}

type DatabaseConfig struct {
	MysqlConfig
	RedisConfig
}

type MysqlConfig struct {
	Username string
	Password string
	Addr     string
	DB       string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}
