package pkg

import (
	log "github.com/Sirupsen/logrus"
	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("SPHINX_HOST", "127.0.0.1")
	viper.SetDefault("SPHINX_PORT", 9306)
	viper.SetDefault("MYSQL_HOST", "127.0.0.1")
	viper.SetDefault("MYSQL_PORT", 3306)
	viper.SetDefault("MYSQL_DBNAME", "hsmppdb")
	viper.SetDefault("MYSQL_USER", "root")
	viper.SetDefault("MYSQL_PASSWORD", "")
	viper.SetDefault("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	viper.SetDefault("RABBITMQ_EXCHANGE", "smppworker-exchange")
	viper.SetDefault("INFLUXDB_ADDR", "http://localhost:8086")
	viper.SetDefault("INFLUXDB_USERNAME", "")
	viper.SetDefault("INFLUXDB_PASSWORD", "")
	viper.SetDefault("HTTP_PORT", 8443)
	viper.SetDefault("HTTP_HOST", "")
	viper.SetDefault("HTTP_CERTFILE", "keys/cert.pem")
	viper.SetDefault("HTTP_KEYFILE", "keys/server.key")
	viper.SetDefault("SOAPSERVICE_HOST", "")
	viper.SetDefault("SOAPSERVICE_PORT", 8445)
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.WithError(err).Warn("Couldn't read config file.")
	}
	viper.SetEnvPrefix("SMPP")
	viper.AutomaticEnv()
}
