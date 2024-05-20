package configuration

import (
	"bankingApp/internal/model"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

func newAppConfiguration() model.IAppConfiguration {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	configFile := fmt.Sprintf("config-%s.yaml", os.Getenv("ENVIRONMENT"))

	cfgFile, err := loadConfig(configFile)
	if err != nil {
		log.Fatalf("loadConfig: %v", err)
	}

	cfg, err := parseConfig(cfgFile)
	if err != nil {
		log.Fatalf("parseConfig: %v", err)
	}
	return cfg
}

func loadConfig(filename string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigName(filename)
	v.AddConfigPath(".")
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			return nil, errors.New("config file not found")
		}
		return nil, err
	}
	return v, nil
}

func parseConfig(v *viper.Viper) (*appConfig, error) {
	var c appConfig
	err := v.Unmarshal(&c)
	if err != nil {
		slog.Info("unable to decode into struct, %v", err) // nolint
		return nil, err
	}
	return &c, nil
}

type appConfig struct {
	GinRunMode     string
	DbUser         string
	DbPass         string
	DbHost         string
	DbName         string
	DbPort         string
	DbMaxOpen      string
	DbMaxIdle      string
	DbMaxTime      string
	DbMaxConn      string
	AppReadTimeout string
	AppServerPort  string
	ThirdPartyAPI  string
	Secret         string
}

func (a *appConfig) ReadTimeout() uint32 {
	return uint32(convertToInt(a.AppReadTimeout))
}

func (a *appConfig) ServerPort() uint32 {
	return uint32(convertToInt(a.AppServerPort))
}

func (a *appConfig) ThirdPartyBaseUrl() string {
	return a.ThirdPartyAPI
}

func (a *appConfig) GinMode() string {
	return a.GinRunMode
}

func (a *appConfig) Username() string {
	return a.DbUser
}

func (a *appConfig) Password() string {
	return a.DbPass
}

func (a *appConfig) Host() string {
	return a.DbHost
}

func (a *appConfig) Port() int {
	return convertToInt(a.DbPort)
}

func (a *appConfig) DatabaseName() string {
	return a.DbName
}

func (a *appConfig) MaximumOpenConnection() int {
	return convertToInt(a.DbMaxOpen)
}

func (a *appConfig) MaximumIdleConnection() int {
	return convertToInt(a.DbMaxConn)
}

func (a *appConfig) MaximumIdleTime() int {
	return convertToInt(a.DbMaxIdle)
}

func (a *appConfig) MaximumTime() int {
	return convertToInt(a.DbMaxTime)
}

func (a *appConfig) JwtSecret() string {
	return a.Secret
}

func convertToInt(valueToBeConverted string) int {
	if val, err := strconv.Atoi(valueToBeConverted); err == nil {
		return val
	}
	return 0
}
