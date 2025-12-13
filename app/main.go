package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	mysqlRepo "github.com/bxcodec/go-clean-arch/internal/repository/mysql"
	myRedisCache "github.com/bxcodec/go-clean-arch/internal/repository/redis"

	"github.com/bxcodec/go-clean-arch/internal/rest"
	"github.com/bxcodec/go-clean-arch/internal/rest/middleware"
	"github.com/bxcodec/go-clean-arch/internal/usecase/article"
	"github.com/bxcodec/go-clean-arch/internal/usecase/user"
	"github.com/joho/godotenv"
)

const (
	defaultTimeout     = 30
	defaultAddress     = ":9090"
	defaultCacheDB     = 0
	dbMaxRetry         = 10
	dbRetryIntervalSec = 2
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	//prepare database
	dbHost := os.Getenv("DATABASE_HOST")
	dbPort := os.Getenv("DATABASE_PORT")
	dbUser := os.Getenv("DATABASE_USER")
	dbPass := os.Getenv("DATABASE_PASS")
	dbName := os.Getenv("DATABASE_NAME")
	connection := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPass, dbHost, dbPort, dbName)
	val := url.Values{}
	val.Add("parseTime", "1")
	val.Add("loc", "Asia/Jakarta")
	dsn := fmt.Sprintf("%s?%s", connection, val.Encode())

	var (
		db  *gorm.DB
		err error
	)

	for i := range dbMaxRetry {
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Printf("failed to open connection to database (attempt %d/%d): %v", i+1, dbMaxRetry, err)
		} else {
			sqlDB, err := db.DB()
			if err != nil {
				log.Printf("failed to get sql.DB from gorm.DB (attempt %d/%d): %v", i+1, dbMaxRetry, err)
				continue
			}
			err = sqlDB.Ping()
			if err == nil {
				break
			}
			log.Printf("failed to ping database (attempt %d/%d): %v", i+1, dbMaxRetry, err)
			_ = sqlDB.Close()
		}

		time.Sleep(dbRetryIntervalSec * time.Second)
	}

	if err != nil {
		log.Fatal("could not connect to database after retries:", err)
	}

	defer func() {
		sqlDB, err := db.DB()
		if err != nil {
			log.Fatal("got error when getting sql.DB from gorm.DB", err)
		}
		if err := sqlDB.Close(); err != nil {
			log.Fatal("got error when closing the DB connection", err)
		}
	}()

	// prepare cache
	cacheHost := os.Getenv("CACHE_HOST")
	cachePort := os.Getenv("CACHE_PORT")
	cachePass := os.Getenv("CACHE_PASS")
	cacheDBStr := os.Getenv("CACHE_DB")
	cacheDB, err := strconv.Atoi(cacheDBStr)
	if err != nil {
		log.Println("failed to parse cacheDB, using default cacheDB")
		cacheDB = defaultCacheDB
	}
	client := redis.NewClient(&redis.Options{
		Addr:     cacheHost + ":" + cachePort,
		Password: cachePass,
		DB:       cacheDB,
	})
	defer func() {
		err = client.Close()
		if err != nil {
			log.Fatal("got error when closing the DB connection", err)
		}
	}()

	_, err = client.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal("failed to open connection to cache", err)
		return
	}

	// prepare gin
	route := gin.Default()
	route.Use(middleware.CORS())
	timeoutStr := os.Getenv("CONTEXT_TIMEOUT")
	timeout, err := strconv.Atoi(timeoutStr)
	if err != nil {
		log.Println("failed to parse timeout, using default timeout")
		timeout = defaultTimeout
	}
	timeoutContext := time.Duration(timeout) * time.Second
	route.Use(middleware.SetRequestContextWithTimeout(timeoutContext))

	// Prepare Repository
	userRepo := mysqlRepo.NewUserRepository(db)
	articleRepo := mysqlRepo.NewArticleRepository(db)
	articleCache := myRedisCache.NewArticleCache(client)

	// Build service Layer
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	jwtTTLStr := os.Getenv("JWT_EXPIRE_HOURS")
	jwtTTL, err := strconv.Atoi(jwtTTLStr)
	if err != nil {
		log.Println("failed to parse JWT TTL, using default 24 hours")
		jwtTTL = 24
	}
	articleSvc := article.NewService(articleRepo, userRepo, articleCache)
	userSvc := user.NewService(userRepo, jwtSecret, time.Duration(jwtTTL)*time.Hour)
	articleHandler := rest.NewArticleHandler(articleSvc)
	userHandler := rest.NewUserHandler(userSvc)

	authMiddleware := middleware.AuthMiddleware(string(jwtSecret))

	// Register routes
	route.POST("/register", userHandler.Register)
	route.POST("/login", userHandler.Login)

	route.GET("/articles", articleHandler.FetchArticle)
	route.GET("/articles/:id", articleHandler.GetByID)

	authorized := route.Group("/")
	authorized.Use(authMiddleware)
	{
		authorized.POST("/articles", articleHandler.Store)
		authorized.DELETE("/articles/:id", articleHandler.Delete)
	}

	// Start Server
	address := os.Getenv("SERVER_ADDRESS")
	if address == "" {
		address = defaultAddress
	}
	log.Fatal(route.Run(address)) //nolint
}
