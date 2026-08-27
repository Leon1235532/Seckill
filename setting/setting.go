package setting

import (
	"os"

	"github.com/joho/godotenv"
	"gopkg.in/ini.v1"
)

var Conf = new(AppConfig)

type AppConfig struct {
	Release      bool `ini:"release"`
	Port         int  `ini:"port"`
	*MySQLConfig `ini:"mysql"`
}

type MySQLConfig struct {
	User     string
	Password string
	DB       string `ini:"db"`
	Host     string `ini:"host"`
	Port     int    `ini:"port"`
}

func Init(file string) error {
	_ = godotenv.Load("./config/.env")
	err := ini.MapTo(Conf, file)
	if err != nil {
		return err
	}
	Conf.MySQLConfig.User = os.Getenv("MYSQL_USER")
	Conf.MySQLConfig.Password = os.Getenv("MYSQL_PASSWORD")
	return nil
}
