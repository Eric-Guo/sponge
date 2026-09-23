package sse

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/Eric-Guo/sponge/pkg/httpcli"
	"github.com/Eric-Guo/sponge/pkg/utils"
)

func runSSEServer(port int, hub *Hub) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	count := 10000
	r.GET("/events", func(c *gin.Context) {
		count++
		uid := strconv.Itoa(count) // mock user id
		hub.Serve(c, uid)
	})

	r.Run(":" + strconv.Itoa(port))
}

func TestServe(t *testing.T) {
	port, _ := utils.GetAvailablePort()
	eventType := DefaultEventType

	gin.SetMode(gin.TestMode)
	r := gin.New()
	hub := NewHub()
	defer hub.Close()
	r.GET("/events", func(c *gin.Context) {
		c.Set("uid", "user1")
		handler := hub.ServeHandler(WithServeExtraHeaders(map[string]string{"X-Test": "test"}))
		handler(c)
	})
	go func() {
		r.Run(":" + strconv.Itoa(port))
	}()

	time.Sleep(100 * time.Millisecond) // wait for server to start

	client := NewClient(fmt.Sprintf("http://localhost:%d/events", port))
	var receivedEvent *Event
	client.OnEvent(eventType, func(event *Event) {
		t.Log("on event", event)
		receivedEvent = event
	})
	err := client.Connect()
	assert.NoError(t, err)
	defer client.Disconnect()

	time.Sleep(100 * time.Millisecond)

	event := &Event{
		Event: eventType,
		Data:  map[string]string{"msg": "hi"},
	}
	_ = hub.Push([]string{"user1"}, event)

	time.Sleep(100 * time.Millisecond) // wait for event to be received

	assert.Equal(t, event.Event, receivedEvent.Event)
}

func TestServeHTTPWithMissingUID(t *testing.T) {
	port, _ := utils.GetAvailablePort()

	hub := NewHub()
	defer hub.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/events", hub.ServeHandler())
	go func() {
		r.Run(":" + strconv.Itoa(port))
	}()
	time.Sleep(100 * time.Millisecond) // wait for server to start

	req, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/events", port), nil)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestPushEventHandler(t *testing.T) {
	port, _ := utils.GetAvailablePort()
	eventType := DefaultEventType

	gin.SetMode(gin.TestMode)
	r := gin.New()
	hub := NewHub()
	defer hub.Close()
	r.GET("/events", func(c *gin.Context) {
		c.Set("uid", "u123")
		handler := hub.ServeHandler()
		handler(c)
	})
	r.POST("/push", hub.PushEventHandler())
	go func() {
		r.Run(":" + strconv.Itoa(port))
	}()

	time.Sleep(100 * time.Millisecond) // wait for server to start

	result := map[string]interface{}{}
	err := httpcli.Post(&result, fmt.Sprintf("http://localhost:%d/push", port), nil)
	assert.Error(t, err)

	event := &Event{
		Event: eventType,
		Data:  "hello world",
	}
	body := PushRequest{
		UIDs:   []string{"u123"},
		Events: []*Event{event},
	}

	result = map[string]interface{}{}
	err = httpcli.Post(&result, fmt.Sprintf("http://localhost:%d/push", port), &body)
	assert.NoError(t, err)
	assert.Equal(t, "ok", result["msg"])
}
