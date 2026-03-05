package main

import (
	"Fuse/fuse"
	"Fuse/middleware"
	"log"
	"net/http"
)

func main() {
	app := fuse.New()

	app.Use(middleware.Defaults()...)

	httpSrv := app.HTTP()
	httpSrv.GET("/ping/:id", func(c fuse.Context) fuse.Result {
		id := c.Param("id")
		log.Printf("id: %s", id)
		return c.Success(fuse.H{"message": "pong"}).WithHttpStatus(http.StatusInternalServerError)
	})

	if err := app.Run(); err != nil {
		panic(err)
	}
}
