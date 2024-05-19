package configuration

import (
	"bankingApp/internal/api/bankservice"
	"bankingApp/internal/api/handlers"
	"bankingApp/internal/api/middleware"
	"bankingApp/internal/model"
	"bankingApp/internal/nethttp"
	"bankingApp/internal/repository"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type App struct {
	DB                  *gorm.DB
	bankTransferHandler *handler.BankTransferHandler
	Configuration       model.IAppConfiguration
	bankTransferService handler.IBankTransferService
}

// NewApp creates a new application instance
func NewApp() *App {
	app := &App{}

	app.Configuration = newAppConfiguration()

	var dbErr error
	app.DB, dbErr = app.connectDatabase(app.Configuration)
	if dbErr != nil {
		panic(dbErr)
	}

	transactionRepository := repository.NewTransactionRepository(app.DB)
	userRepository := repository.NewUserRepository(app.DB)
	accountRepository := repository.NewAccountRepository(app.DB)

	timeout := app.Configuration.ReadTimeout()
	duration := time.Duration(time.Duration.Seconds(time.Duration(timeout)))
	restClient := nethttp.NewRestHttpClient(duration)

	app.bankTransferService = bankservice.NewBankService(
		app.Configuration,
		transactionRepository,
		userRepository,
		accountRepository,
		restClient)
	app.bankTransferHandler = handler.NewBankTransferHandler(app.bankTransferService)
	return app
}

// connectDatabase sets up a DB connection configuration
func (app *App) connectDatabase(config model.IAppConfiguration) (*gorm.DB, error) {
	// Get the values from the config struct
	user := config.Username()
	password := config.Password()
	host := config.Host()
	port := config.Port()
	dbName := config.DatabaseName()

	// Construct the connection string
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True", user, password, host, port, dbName) // nolint
	log.Println(dsn)

	// Open a connection to the database
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{NamingStrategy: schema.NamingStrategy{SingularTable: true, TablePrefix: "tbl_"}})
	if err != nil {
		log.Fatal(err.Error())
		return nil, err
	}

	db.Logger = logger.Default.LogMode(logger.Info)

	dbConfig, err := db.DB()
	if err != nil {
		return nil, err
	}

	dbConfig.SetMaxOpenConns(config.MaximumOpenConnection())
	dbConfig.SetConnMaxIdleTime(time.Duration(config.MaximumIdleTime()) * time.Second)
	dbConfig.SetConnMaxLifetime(time.Duration(config.MaximumTime()) * time.Second)
	dbConfig.SetMaxIdleConns(config.MaximumIdleConnection())
	return db, nil
}

// RouteHandler sets up the application routes and middleware
func (app *App) RouteHandler(config model.IAppConfiguration) *gin.Engine {
	route := gin.Default()
	gin.SetMode(config.GinMode())
	route.HandleMethodNotAllowed = true
	route.NoMethod(handler.NoMethodHandler)
	route.NoRoute(handler.NotFoundHandler)

	groupRoute := route.Group("/api/v1/bank")

	loggingMiddleware := middleware.LoggingMiddleware{}
	groupRoute.Use(loggingMiddleware.RequestLogger())
	groupRoute.Use(loggingMiddleware.ResponseLogger())

	securityMiddleware := middleware.SecurityMiddleware{}
	groupRoute.Use(securityMiddleware.RequestHeaders())

	groupRoute.POST("/fund-transfer", app.bankTransferHandler.Transfer)
	groupRoute.GET("/status-query/:ref", app.bankTransferHandler.StatusQuery)
	return route
}
