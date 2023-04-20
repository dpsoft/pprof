package main

import (
	"github.com/dpsoft/pprof"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	pprof.Register(router)
	router.Run(":8080")
}
