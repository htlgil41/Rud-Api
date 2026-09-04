package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type RootConfig struct {
	Config Config `mapstructure:"config"`
}

type Config struct {
	DB       DBConfig       `mapstructure:"db"`
	Server   ServerConfig   `mapstructure:"server"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Rabbit   RabbitConfig   `mapstructure:"rabbit"`
	Reportes ReportesConfig `mapstructure:"reportes"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
}

type RabbitConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	Exchange     string `mapstructure:"exchange"`
	ExchangeType string `mapstructure:"exchange_type"`
	Consumer     string `mapstructure:"consumer"`
}

type ReportesConfig struct {
	RoutingKey string `mapstructure:"routing_key"`
}

type DBConfig struct {
	PGDBRud       PGDBRudConfig       `mapstructure:"pgdbrud"`
	CorporativoDB CorporativoDBConfig `mapstructure:"corporativodb"`
}

type PGDBRudConfig struct {
	Host     string `mapstructure:"host"`
	Port     int32  `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DB       string `mapstructure:"db"`
}

type CorporativoDBConfig struct {
	URI string `mapstructure:"uri"`
}

type ServerConfig struct {
	Mode string `mapstructure:"mode"`
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

func LoadConfigWithVyper() *Config {
	var config RootConfig
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.SetConfigFile("")
	viper.AddConfigPath(".")

	if errReadConfig := viper.ReadInConfig(); errReadConfig != nil {
		return nil
	}
	if initConfig := viper.Unmarshal(&config); initConfig != nil {
		return nil
	}

	fmt.Println("Load Configuration with viper")
	return &config.Config
}
