package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	mysqlRepo "github.com/bxcodec/go-clean-arch/internal/repository/mysql"
	myRedisCache "github.com/bxcodec/go-clean-arch/internal/repository/redis"

	"github.com/bxcodec/go-clean-arch/article"
	"github.com/bxcodec/go-clean-arch/internal/rest"
	"github.com/bxcodec/go-clean-arch/internal/rest/middleware"
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
		dbConn *sql.DB
		err    error
	)

	for i := 0; i < dbMaxRetry; i++ {
		dbConn, err = sql.Open(`mysql`, dsn)
		if err != nil {
			log.Printf("failed to open connection to database (attempt %d/%d): %v", i+1, dbMaxRetry, err)
		} else {
			err = dbConn.Ping()
			if err == nil {
				break
			}
			log.Printf("failed to ping database (attempt %d/%d): %v", i+1, dbMaxRetry, err)
			_ = dbConn.Close()
		}

		time.Sleep(dbRetryIntervalSec * time.Second)
	}

	if err != nil {
		log.Fatal("could not connect to database after retries:", err)
	}

	defer func() {
		if err := dbConn.Close(); err != nil {
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

	// prepare echo
	e := echo.New()
	e.Use(middleware.CORS)
	timeoutStr := os.Getenv("CONTEXT_TIMEOUT")
	timeout, err := strconv.Atoi(timeoutStr)
	if err != nil {
		log.Println("failed to parse timeout, using default timeout")
		timeout = defaultTimeout
	}
	timeoutContext := time.Duration(timeout) * time.Second
	e.Use(middleware.SetRequestContextWithTimeout(timeoutContext))

	// Prepare Repository
	authorRepo := mysqlRepo.NewAuthorRepository(dbConn)
	articleRepo := mysqlRepo.NewArticleRepository(dbConn)
	articleCache := myRedisCache.NewArticleCache(client)

	// Build service Layer
	svc := article.NewService(articleRepo, authorRepo, articleCache)
	rest.NewArticleHandler(e, svc)

	// Start Server
	address := os.Getenv("SERVER_ADDRESS")
	if address == "" {
		address = defaultAddress
	}
	log.Fatal(e.Start(address)) //nolint
}
