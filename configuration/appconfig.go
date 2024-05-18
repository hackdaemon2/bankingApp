package configuration

import (
	"bankingApp/internal/model"
	"strconv"
)

func NewAppConfiguration(envMap map[string]string) model.IAppConfiguration {
	return &appConfig{
		GinRunMode:     envMap["GIN_MODE"],
		DbUser:         envMap["DB_USER"],
		DbPass:         envMap["DB_PASSWORD"],
		DbHost:         envMap["DB_HOST"],
		DbName:         envMap["DB_NAME"],
		DbPort:         envMap["DB_PORT"],
		DbMaxOpen:      envMap["DB_MAX_OPEN"],
		DbMaxIdle:      envMap["DB_MAX_IDLE_TIME"],
		DbMaxTime:      envMap["DB_MAX_TIME"],
		DbMaxConn:      envMap["DB_MAX_IDLE_CONN"],
		AppReadTimeout: envMap["APP_READ_TIMEOUT"],
		AppServerPort:  envMap["APP_SERVER_PORT"],
		ThirdPartyAPI:  envMap["THIRD_PARTY_API"],
		Secret:         envMap["APP_JWT_SECRET"],
	}
}

type appConfig struct {
	GinRunMode     string `env:"GIN_MODE"`
	DbUser         string `env:"DB_USER"`
	DbPass         string `env:"DB_PASSWORD"`
	DbHost         string `env:"DB_HOST"`
	DbName         string `env:"DB_NAME"`
	DbPort         string `env:"DB_PORT"`
	DbMaxOpen      string `env:"DB_MAX_OPEN"`
	DbMaxIdle      string `env:"DB_MAX_IDLE_TIME"`
	DbMaxTime      string `env:"DB_MAX_TIME"`
	DbMaxConn      string `env:"DB_MAX_IDLE_CONN"`
	AppReadTimeout string `env:"APP_READ_TIMEOUT"`
	AppServerPort  string `env:"APP_SERVER_PORT"`
	ThirdPartyAPI  string `env:"THIRD_PARTY_API"`
	Secret         string `env:"APP_JWT_SECRET"`
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
