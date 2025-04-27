package config

type Config struct {
	HTTP      HTTP      `yaml:"http"`
	Backends  []Backend `yaml:"backends"`
	DB        DB        `yaml:"postgres"`
	RateLimit RateLimit `yaml:"ratelimit"`
}

type HTTP struct {
	Host string `yaml:"host" env-required:"true" env:"HTTP_HOST"`
	Port string `yaml:"port" env-required:"true" env:"HTTP_PORT"`
}

type Backend struct {
	Host string `yaml:"host" env-required:"true" env:"BACKEND_HOST"`
	Port string `yaml:"port" env-required:"true" env:"BACKEND_PORT"`
}

type DB struct {
	Host     string `yaml:"host" env-required:"true" env:"PG_HOST"`
	Port     string `yaml:"port" env-required:"true" env:"PG_PORT"`
	User     string `yaml:"user" env-required:"true" env:"PG_USER"`
	Password string `yaml:"password" env-required:"true" env:"PG_PASSWORD"`
	DBName   string `yaml:"name" env:"PG_NAME" env-required:"true" `
	PgDriver string `yaml:"pg_driver" env:"PG_PG_DRIVER" env-required:"true" `
	Schema   string `yaml:"schema" env:"PG_SCHEMA" env-required:"true" `
}

type RateLimit struct {
	DefaultCapacity       int64 `yaml:"default_capacity"`
	DefaultRefillRate     int64 `yaml:"default_refill_rate"`
	RefillIntervalSeconds int   `yaml:"refill_interval_seconds"`
}
