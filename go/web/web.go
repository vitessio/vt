package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/vitessio/vt/go/summarize"
)

func RenderFileToGin(fileName string, data any, c *gin.Context) {
	buf, err := RenderFile(fileName, data)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
}

func RenderFile(fileName string, data any) (*bytes.Buffer, error) {
	_ = data
	tmpl := template.Must(template.ParseFiles(
		"go/web/templates/layout.html",
		"go/web/templates/footer.html",
		"go/web/templates/header.html",
		fmt.Sprintf("go/web/templates/%s", fileName),
	))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "layout.html", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %v", err)
	}
	return &buf, nil
}

func Run() {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard // Disable logging
	r := gin.Default()

	r.LoadHTMLGlob("go/web/templates/*.html")
	r.Static("/css", "go/web/templates/css")
	r.Static("/images", "go/web/templates/images")

	r.GET("/", func(c *gin.Context) {
		RenderFileToGin("index.html", nil, c)
	})

	r.GET("/about", func(c *gin.Context) {
		RenderFileToGin("about.html", nil, c)
	})

	r.GET("/summarize", func(c *gin.Context) {
		filePath := c.Query("file")
		data, err := os.ReadFile(filePath)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		var summary summarize.Summary
		err = json.Unmarshal(data, &summary)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		RenderFileToGin("summarize.html", summary, c)
	})

	if os.WriteFile("/dev/stderr", []byte("Starting web server on http://localhost:8080\n"), 0o600) != nil {
		panic("Failed to write to /dev/stderr")
	}
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
