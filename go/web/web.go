package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/vitessio/vt/go/summarize"
	"github.com/vitessio/vt/go/web/state"
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
	tmpl := template.Must(template.New("summarize.html").Funcs(getFuncMap()).ParseFiles(
		"go/web/templates/layout.html",
		"go/web/templates/footer.html",
		"go/web/templates/header.html",
		fmt.Sprintf("go/web/templates/%s", fileName),
	))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "layout.html", data)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %v", err)
	}
	return &buf, nil
}

type SummaryOutput struct {
	summarize.Summary
	DateOfAnalysis string
}

func getFuncMap() template.FuncMap {
	return template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"divide": func(a, b any) float64 {
			if b == 0 || b == nil {
				return 0 // Handle division by zero or nil
			}

			// Convert `a` and `b` to float64
			var aFloat, bFloat float64

			switch v := a.(type) {
			case int:
				aFloat = float64(v)
			case float64:
				aFloat = v
			default:
				return 0 // Invalid type, return 0
			}

			switch v := b.(type) {
			case int:
				bFloat = float64(v)
			case float64:
				bFloat = v
			default:
				return 0 // Invalid type, return 0
			}

			return aFloat / bFloat
		},
	}
}

func addFuncMap(r *gin.Engine) {
	r.SetFuncMap(getFuncMap())
}

func Run(s *state.State) error {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard // Disable logging
	r := gin.Default()
	addFuncMap(r)

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
			fmt.Printf("Error unmarshalling summary: %v\n", err)
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		summarizeOutput := SummaryOutput{
			Summary:        summary,
			DateOfAnalysis: time.Now().Format(time.DateTime),
		}

		RenderFileToGin("summarize.html", &summarizeOutput, c)
	})

	if _, err := fmt.Fprintf(os.Stderr, "Starting web server on http://localhost:%d\n", s.GetPort()); err != nil {
		return err
	}

	if err := r.Run(fmt.Sprintf(":%d", s.GetPort())); err != nil {
		return err
	}
	return nil
}
