package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/Guyuepp/Go-Clean-Architecture-Blog/internal/repository"
	mysqlRepo "github.com/Guyuepp/Go-Clean-Architecture-Blog/internal/repository/mysql"
	myRedisCache "github.com/Guyuepp/Go-Clean-Architecture-Blog/internal/repository/redis"
	"github.com/Guyuepp/Go-Clean-Architecture-Blog/internal/workers"

	"github.com/Guyuepp/Go-Clean-Architecture-Blog/internal/rest"
	"github.com/Guyuepp/Go-Clean-Architecture-Blog/internal/rest/middleware"
	"github.com/Guyuepp/Go-Clean-Architecture-Blog/internal/usecase/article"
	"github.com/Guyuepp/Go-Clean-Architecture-Blog/internal/usecase/comment"
	"github.com/Guyuepp/Go-Clean-Architecture-Blog/internal/usecase/user"
	"github.com/joho/godotenv"
)

const (
	defaultTimeout      = 30
	defaultAddress      = ":9090"
	defaultCacheDB      = 0
	defaultBloomBitSize = 10000000
	dbMaxRetry          = 10
	dbRetryIntervalSec  = 2
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
	commentRepo := mysqlRepo.NewCommentRepository(db)

	// Article相关的三层架构
	// 1. DB层
	articleDBRepo := mysqlRepo.NewArticleDBRepository(db)
	// 2. Cache层
	articleCache := myRedisCache.NewArticleCache(client)
	// 3. Repository协调层
	articleRepo := repository.NewArticleRepository(articleDBRepo, articleCache, userRepo)

	bloomBitSizeStr := os.Getenv("BLOOM_FILTER_SIZE")
	bloomBitSize, err := strconv.ParseUint(bloomBitSizeStr, 10, 64)
	if err != nil {
		log.Printf("failed to parse bloom bit size, using default size")
		bloomBitSize = defaultBloomBitSize
	}
	bloomRepo := myRedisCache.NewRedisBloomRepo(client, bloomBitSize)

	// Start worker
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	views_syncer := workers.NewSyncViewWorker(articleDBRepo, articleCache)
	go views_syncer.Start(ctx)

	likes_syncer := workers.NewSyncLikesWorker(articleDBRepo)
	go likes_syncer.Start(ctx)

	// Build service Layer
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	jwtTTLStr := os.Getenv("JWT_EXPIRE_HOURS")
	jwtTTL, err := strconv.Atoi(jwtTTLStr)
	if err != nil {
		log.Println("failed to parse JWT TTL, using default 24 hours")
		jwtTTL = 24
	}
	// usecase层只依赖repository接口和cache（用于点赞等特殊操作）
	articleSvc := article.NewService(articleRepo, articleCache, likes_syncer, bloomRepo)
	userSvc := user.NewService(userRepo, jwtSecret, time.Duration(jwtTTL)*time.Hour)
	commentSvc := comment.NewService(commentRepo, bloomRepo)
	articleHandler := rest.NewArticleHandler(articleSvc)
	userHandler := rest.NewUserHandler(userSvc)
	commentHandler := rest.NewCommentHandler(commentSvc)

	authMiddleware := middleware.AuthMiddleware(string(jwtSecret))

	// Prepare bloom filter
	if err := articleSvc.InitBloomFilter(ctx); err != nil {
		log.Printf("failed to init bloom filter: %v\n", err)
		return
	}

	// Register routes
	route.POST("/register", userHandler.Register)
	route.POST("/login", userHandler.Login)

	route.GET("/articles", articleHandler.FetchArticle)
	route.GET("/articles/:id", articleHandler.GetByID)

	route.GET("/articles/ranks", articleHandler.FetchRank)

	route.GET("/articles/:id/comments", commentHandler.FetchCommentsByArticle)

	authorized := route.Group("/")
	authorized.Use(authMiddleware)
	{
		authorized.POST("/articles", articleHandler.Store)
		authorized.DELETE("/articles/:id", articleHandler.Delete)
		authorized.POST("/articles/:id/like", articleHandler.Like)
		authorized.DELETE("/articles/:id/like", articleHandler.Unlike)
		authorized.POST("/articles/:id/comments", commentHandler.CreateComment)
		authorized.DELETE("/articles/:id/comments", commentHandler.DeleteComment)
	}

	// Start Server
	address := os.Getenv("SERVER_ADDRESS")
	if address == "" {
		address = defaultAddress
	}
	srv := &http.Server{
		Addr:    address,
		Handler: route,
	}
	go func() {
		log.Printf("Server is running on %s\n", address)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err) // nolint
		}
	}()

	// shutdown
	<-ctx.Done()
	log.Println("Shutdown signal received, stopping server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Waiting for worker to cleanup...")
	time.Sleep(2 * time.Second)

	log.Println("Server exiting")
}
