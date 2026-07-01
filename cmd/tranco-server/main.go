package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/WangYihang/tranco-go-package"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/singleflight"
)

// maxCachedLists bounds how many distinct dates' lists are kept in memory
// at once. Each entry can hold millions of rows, so this caps a
// long-running server's memory use rather than growing it forever.
const maxCachedLists = 16

// shutdownTimeout bounds how long the server waits for in-flight requests
// to finish when asked to shut down before forcing the shutdown anyway.
const shutdownTimeout = 10 * time.Second

// newRouter builds the server's route table. Split out from main so tests
// can exercise it directly via httptest without binding a real port.
func newRouter(trancoLists *trancoListCache) *gin.Engine {
	var listGroup singleflight.Group

	r := gin.Default()
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.GET("/rank/:domain/date/:date", func(c *gin.Context) {
		domain := c.Param("domain")
		date := c.Param("date")

		if _, err := time.Parse("2006-01-02", date); err != nil {
			c.JSON(400, gin.H{
				"message": "invalid date, expected format YYYY-MM-DD",
				"status":  "error",
			})
			return
		}

		list, ok := trancoLists.get(date)

		if !ok {
			// singleflight collapses concurrent requests for the same new
			// date into one download, and - critically - runs it without
			// holding the cache lock, so a slow first-time download for one
			// date no longer blocks requests for other, already-cached
			// dates.
			v, err, _ := listGroup.Do(date, func() (interface{}, error) {
				newList, err := tranco.NewTrancoList(date, true, "full", ".tranco", tranco.WithQuiet())
				if err != nil {
					return nil, err
				}
				trancoLists.set(date, newList)
				return newList, nil
			})
			if err != nil {
				c.JSON(500, gin.H{
					"message": "error occured while obtaining tranco list",
					"status":  "error",
				})
				return
			}
			list = v.(*tranco.TrancoList)
		}

		rank, err := list.Rank(domain)
		switch {
		case errors.Is(err, tranco.ErrDomainNotFound):
			c.JSON(404, gin.H{
				"message": "domain not found in tranco list",
				"status":  "error",
			})
			return
		case err != nil:
			c.JSON(500, gin.H{
				"message": "error occured while querying rank",
				"status":  "error",
			})
			return
		}
		c.JSON(200, gin.H{
			"message": "success",
			"status":  "success",
			"rank":    rank,
		})
	})

	return r
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := newRouter(newTrancoListCache(maxCachedLists))

	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s", err)
		}
	}()

	// Wait for SIGINT/SIGTERM (e.g. from `docker stop` or a k8s pod
	// eviction) and let in-flight requests finish instead of killing them
	// mid-response.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shut down: %s", err)
	}
}
