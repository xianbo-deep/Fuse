package main

import (
	"Fuse/fuse"
	"Fuse/middleware"
	"log"
)

func main() {
	app := fuse.New()

	app.Use(middleware.Defaults()...)

	httpSrv := app.HTTP()
	httpSrv.Get("/ping/:id", func(c fuse.Context) fuse.Result {
		id := c.Param("id")
		log.Printf("id: %s", id)
		return c.Success(fuse.H{"message": "pong"})
	})

	if err := app.Run(":8080"); err != nil {
		panic(err)
	}
}
