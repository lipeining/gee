package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"
	"web/engine"
)

func trace(message string) string {
	var pcs [32]uintptr
	n := runtime.Callers(3, pcs[:]) // skip first 3 caller

	var str strings.Builder
	str.WriteString(message + "\nTraceback:")
	for _, pc := range pcs[:n] {
		fn := runtime.FuncForPC(pc)
		file, line := fn.FileLine(pc)
		str.WriteString(fmt.Sprintf("\n\t%s:%d", file, line))
	}
	return str.String()
}

func Recovery() engine.HandlerFunc {
	return func(c *engine.Context) {
		defer func() {
			if err := recover(); err != nil {
				message := fmt.Sprintf("%s", err)
				log.Printf("%s\n\n", trace(message))
				c.Fail(http.StatusInternalServerError, "Internal Server Error")
			}
		}()

		c.Next()
	}
}

func Logger() engine.HandlerFunc {
	return func(c *engine.Context) {
		t := time.Now()
		c.Next()
		// Calculate resolution time
		log.Printf("[%d] %s in %v", c.StatusCode, c.Req.RequestURI, time.Since(t))
	}
}

func main() {
	e := engine.New()

	e.Use(Recovery())
	e.Get("/panic", func(c *engine.Context) {
		names := []string{"hello"}
		c.String(http.StatusOK, names[100])
	})

	e.Get("/hello", func(c *engine.Context) {
		c.HTML(http.StatusOK, "hello world")
	})

	e.Get("/blogs/:id", func(c *engine.Context) {
		c.HTML(http.StatusOK, "blog:"+c.Param("id"))
	})

	v1 := e.Group("/v1")
	v1.Use(Logger())
	{
		v1.Get("/", func(c *engine.Context) {
			c.HTML(http.StatusOK, "v1 index")
		})

		v1.Get("/hello", func(c *engine.Context) {
			c.String(http.StatusOK, "v1 hello world")
		})
	}

	v2 := e.Group("/v2")
	{
		v2.Get("/hello/:name", func(c *engine.Context) {
			c.String(http.StatusOK, "hello %s", c.Param("name"))
		})
		v2.Post("/login", func(c *engine.Context) {
			c.JSON(http.StatusOK, engine.H{
				"username": c.PostForm("username"),
				"password": c.PostForm("password"),
			})
		})
	}

	e.Run(":3000")
}
