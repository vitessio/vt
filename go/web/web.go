/*
Copyright 2024 The Vitess Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package web

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/vitessio/vt/go/utils"
)

func RenderFileToGin(fileName string, data any, c *gin.Context) {
	buf, err := utils.RenderFile(fileName, "layout.html", data)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
}

func Run(port int64) {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard // Disable logging
	r := gin.Default()
	r.SetFuncMap(utils.GetFuncMap())

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
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	if os.WriteFile("/dev/stderr", []byte(fmt.Sprintf("Starting web server on http://localhost:%d\n", port)), 0o600) != nil {
		panic("Failed to write to /dev/stderr")
	}
	if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
