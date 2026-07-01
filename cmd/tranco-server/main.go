package main

import (
	"errors"
	"sync"
	"time"

	"github.com/WangYihang/tranco-go-package"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/singleflight"
)

func main() {
	trancoLists := make(map[string]*tranco.TrancoList)
	var trancoListsMu sync.Mutex
	var listGroup singleflight.Group

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
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

		trancoListsMu.Lock()
		list, ok := trancoLists[date]
		trancoListsMu.Unlock()

		if !ok {
			// singleflight collapses concurrent requests for the same new
			// date into one download, and - critically - runs it without
			// holding trancoListsMu, so a slow first-time download for one
			// date no longer blocks requests for other, already-cached
			// dates.
			v, err, _ := listGroup.Do(date, func() (interface{}, error) {
				newList, err := tranco.NewTrancoList(date, true, "full", ".tranco", tranco.WithQuiet())
				if err != nil {
					return nil, err
				}
				trancoListsMu.Lock()
				trancoLists[date] = newList
				trancoListsMu.Unlock()
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
	r.Run()
}
