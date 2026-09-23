// Package handlerfunc is used for public http request handler.
package handlerfunc

import (
	"embed"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Eric-Guo/sponge/pkg/errcode"
	"github.com/Eric-Guo/sponge/pkg/utils"
)

// CheckHealthReply check health result
type CheckHealthReply struct {
	Status   string `json:"status"`
	Hostname string `json:"hostname"`
}

// CheckHealth check healthy.
// @Summary check system health status
// @Description Returns system health information including status and hostname
// @Tags system
// @Accept  json
// @Produce  json
// @Success 200 {object} CheckHealthReply "Returns health status information"
// @Router /health [get]
func CheckHealth(c *gin.Context) {
	c.JSON(http.StatusOK, CheckHealthReply{Status: "UP", Hostname: utils.GetHostname()})
}

type PingReply struct{}

// Ping the server
// @Summary ping the server
// @Description Simple ping endpoint to check if server is responsive
// @Tags system
// @Accept  json
// @Produce  json
// @Success 200 {object} PingReply "Returns empty JSON object"
// @Router /ping [get]
func Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// ListCodes list error codes info
// @Summary list all error codes
// @Description Returns a list of all defined HTTP error codes and their descriptions
// @Tags system
// @Accept  json
// @Produce  json
// @Success 200 {array} errcode.ErrInfo "List of error codes"
// @Router /codes [get]
func ListCodes(c *gin.Context) {
	c.JSON(http.StatusOK, errcode.ListHTTPErrCodes())
}

// BrowserRefresh solve vue using history route 404 problem, for system file
func BrowserRefresh(path string) func(c *gin.Context) {
	return func(c *gin.Context) {
		accept := c.Request.Header.Get("Accept")
		flag := strings.Contains(accept, "text/html")
		if flag {
			content, err := os.ReadFile(path)
			if err != nil {
				c.Writer.WriteHeader(404)
				_, _ = c.Writer.WriteString("Not Found")
				return
			}
			c.Writer.WriteHeader(200)
			c.Writer.Header().Add("Accept", "text/html")
			_, _ = c.Writer.Write(content)
			c.Writer.Flush()
		}
	}
}

// BrowserRefreshFS solve vue using history route 404 problem, for embed.FS
func BrowserRefreshFS(fs embed.FS, path string) func(c *gin.Context) {
	return func(c *gin.Context) {
		accept := c.Request.Header.Get("Accept")
		flag := strings.Contains(accept, "text/html")
		if flag {
			content, err := fs.ReadFile(path)
			if err != nil {
				c.Writer.WriteHeader(404)
				_, _ = c.Writer.WriteString("Not Found")
				return
			}
			c.Writer.WriteHeader(200)
			c.Writer.Header().Add("Accept", "text/html")
			_, _ = c.Writer.Write(content)
			c.Writer.Flush()
		}
	}
}
