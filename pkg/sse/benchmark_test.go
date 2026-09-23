package sse

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Eric-Guo/sponge/pkg/utils"
)

func BenchmarkSSEPushOneClient(b *testing.B) {
	port, _ := utils.GetAvailablePort()
	eventType := DefaultEventType
	ctx, cancel := context.WithCancel(context.Background())

	// start sse server
	hub := NewHub(WithContext(ctx, cancel))

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	uid := "u001"
	r.GET("/events", func(c *gin.Context) {
		hub.Serve(c, uid)
	})

	go func() {
		r.Run(":" + strconv.Itoa(port))
	}()

	time.Sleep(100 * time.Millisecond)

	var received int64

	// run sse client
	client := NewClient(fmt.Sprintf("http://localhost:%d/events", port), WithClientReconnectTimeInterval(100*time.Millisecond))
	var wg sync.WaitGroup
	client.OnEvent(eventType, func(e *Event) {
		atomic.AddInt64(&received, 1)
		wg.Done()
	})
	err := client.Connect()
	if err != nil {
		b.Fatalf("Failed to connect SSE client: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	uids := []string{uid}
	var event = &Event{
		Event: eventType,
		Data:  "test-push-data",
	}

	wg.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hub.Push(uids, event) // push event to one client
	}
	b.StopTimer()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// wait for all events to be received
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		b.Error("timeout: SSE client did not receive all events")
	}

	// Reporting metrics (events per second)
	elapsed := b.Elapsed()
	eventsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(eventsPerSec, "events/sec")
}

func BenchmarkSSEServerBroadcast(b *testing.B) {
	port, _ := utils.GetAvailablePort()
	eventType := DefaultEventType
	ctx, cancel := context.WithCancel(context.Background())

	// start sse server
	hub := NewHub(WithContext(ctx, cancel))

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	var count int64 = 10000
	r.GET("/events", func(c *gin.Context) {
		atomic.AddInt64(&count, 1)
		uid := fmt.Sprintf("%d", atomic.LoadInt64(&count)) // mock user id
		hub.Serve(c, uid)
	})
	go func() {
		r.Run(":" + strconv.Itoa(port))
	}()

	time.Sleep(100 * time.Millisecond)

	var received int64
	var wg sync.WaitGroup

	// run sse clients
	clientNum := 10
	for i := 0; i < clientNum; i++ {
		client := NewClient(fmt.Sprintf("http://localhost:%d/events", port), WithClientReconnectTimeInterval(100*time.Millisecond))
		client.OnEvent(eventType, func(e *Event) {
			atomic.AddInt64(&received, 1)
			wg.Done()
		})
		err := client.Connect()
		if err != nil {
			b.Fatalf("Failed to connect SSE client: %v", err)
		}
	}
	time.Sleep(200 * time.Millisecond)

	var event = &Event{
		Event: eventType,
		Data:  "test-push-data",
	}

	wg.Add(b.N * clientNum)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hub.Push(nil, event) // push event to all clients
	}
	b.StopTimer()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// wait for all events to be received
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		b.Error("timeout: SSE client did not receive all events")
	}

	total, success, failed, _ := hub.PushStats.Snapshot()

	// Reporting metrics (events per second)
	elapsed := b.Elapsed()
	eventsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(eventsPerSec, "events/sec")
	b.ReportMetric(float64(total), "total_push")
	b.ReportMetric(float64(success), "success_push")
	b.ReportMetric(float64(failed), "failed_push")
}
