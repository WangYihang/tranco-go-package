package main

import (
	"github.com/WangYihang/tranco-go-package"
	"github.com/gin-gonic/gin"
)

func main() {
	trancoLists := make(map[string]*tranco.TrancoList)
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.GET("/rank/:domain/date/:date", func(c *gin.Context) {
		domain := c.Param("domain")
		date := c.Param("date")
		list, ok := trancoLists[date]
		if !ok {
			var err error
			list, err = tranco.NewTrancoList(date, true, "full")
			if err != nil {
				c.JSON(500, gin.H{
					"message": "error occured while parsing date",
					"status":  "error",
				})
				return
			}
			trancoLists[date] = list
		}
		rank, err := list.Rank(domain)
		if err != nil {
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
