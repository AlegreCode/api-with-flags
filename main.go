package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var (
	redisClient *redis.Client
	limiter     *redis_rate.Limiter
)

func initClients() {
	redisClient = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URL"),
	})
	limiter = redis_rate.NewLimiter(redisClient)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Loading environment variable from the host system")
	} else {
		log.Printf("Loading environment from .env file")
	}

	initClients()
	defer redisClient.Close()

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		remainingLimit, err := rateLimitCall(c.ClientIP())
		if err != nil {
			c.JSON(
				http.StatusTooManyRequests,
				gin.H{"error": "Rate Limit Hit"})
		} else {
			c.JSON(
				http.StatusOK,
				gin.H{"Your left over API request is": remainingLimit})
		}
	})
	r.GET("/beta", func(c *gin.Context) {
		c.JSON(
			http.StatusOK,
			gin.H{"message": "This is beta endpoint"})
	})
	r.Run(":" + os.Getenv("PORT"))
}

func rateLimitCall(ClientIP string) (int, error) {
	ctx := context.Background()

	rateLimitString := os.Getenv("RATE_LIMIT")
	RATE_LIMIT, _ := strconv.Atoi(rateLimitString)

	res, err := limiter.Allow(ctx, ClientIP, redis_rate.PerHour(RATE_LIMIT))
	if err != nil {
		panic(err)
	}

	if res.Remaining == 0 {
		return 0, errors.New("You have hit the Rate Limit for the API. Try again later")
	}

	fmt.Println("remaining request for", ClientIP, "is", res.Remaining)
	return res.Remaining, nil
}
