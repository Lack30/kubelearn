package app

type Config struct {
	Address string   `yaml:"address"`
	DB      DBConfig `yaml:"db"`
}

type DBConfig struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
}
