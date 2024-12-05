package web

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func renderFile(fileName string, c *gin.Context) {
	tmpl := template.Must(template.ParseFiles(
		"go/web/templates/layout.html",
		"go/web/templates/footer.html",
		"go/web/templates/header.html",
		fmt.Sprintf("go/web/templates/%s", fileName),
	))

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "layout.html", nil); err != nil {
		// Return an error response if template execution fails
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	// Set the Content-Type to text/html and write the rendered content
	c.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
}

func Run() {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard // Disable logging
	r := gin.Default()

	r.LoadHTMLGlob("go/web/templates/*.html")
	r.Static("/css", "go/web/templates/css")
	r.Static("/images", "go/web/templates/images")

	r.GET("/", func(c *gin.Context) {
		renderFile("index.html", c)
	})

	r.GET("/about", func(c *gin.Context) {
		renderFile("about.html", c)
	})

	r.GET("/render", func(c *gin.Context) {
		action := c.Query("action")
		param := c.Query("param")

		// Call the handler to process the action
		tmpl := template.Must(template.ParseFiles("go/web/templates/layout.html", "go/web/templates/index.html", "go/web/templates/footer.html", "go/web/templates/header.html"))

		data, err := handleRenderAction(tmpl, action, param)
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		// Return the rendered HTML
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	r.POST("/render", func(c *gin.Context) {
		var input struct {
			Action string `json:"action"`
			Param  string `json:"param"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}
		tmpl := template.Must(template.ParseFiles("go/web/templates/layout.html", "go/web/templates/index.html", "go/web/templates/footer.html", "go/web/templates/header.html"))

		data, err := handleRenderAction(tmpl, input.Action, input.Param)
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})
	if os.WriteFile("/dev/stderr", []byte("Starting web server on http://localhost:8080\n"), 0o600) != nil {
		panic("Failed to write to /dev/stderr")
	}
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func handleRenderAction(tmpl *template.Template, action, param string) ([]byte, error) {
	var data struct {
		Title string
		Body  string
	}

	switch action {
	case "home":
		data.Title = "Home Page"
		data.Body = "Welcome to the homepage!"
	case "about":
		data.Title = "About Page"
		data.Body = fmt.Sprintf("About us: %s", param)
	case "dynamic":
		data.Title = "Dynamic Content"
		data.Body = generateDynamicContent(param)
	default:
		return nil, fmt.Errorf("invalid action: %s", action)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "layout", data); err != nil {
		return nil, fmt.Errorf("failed to render template: %v", err)
	}

	return buf.Bytes(), nil
}

func generateDynamicContent(param string) string {
	if param == "" {
		return "No parameter provided for dynamic content."
	}
	return fmt.Sprintf("Generated dynamic content with param: %s", param)
}
