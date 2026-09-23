// Package stat provides for counting system and process cpu and memory information, alarm notification support.
package stat

import (
	"context"
	"math"
	"runtime"
	"time"

	"go.uber.org/zap"

	"github.com/Eric-Guo/sponge/pkg/stat/cpu"
	"github.com/Eric-Guo/sponge/pkg/stat/mem"
)

var (
	printInfoInterval = time.Minute // minimum 1 second
	zapLog, _         = zap.NewProduction()

	notifyCh = make(chan struct{})
)

// Option set the stat options field.
type Option func(*options)

type options struct {
	enableAlarm   bool
	zapFields     []zap.Field
	customHandler func(ctx context.Context, sd *StatData) error
}

func (o *options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

// WithPrintInterval set print interval
func WithPrintInterval(d time.Duration) Option {
	return func(o *options) {
		if d < time.Second {
			return
		}
		printInfoInterval = d
	}
}

// WithLog set zapLog
func WithLog(l *zap.Logger) Option {
	return func(o *options) {
		if l == nil {
			return
		}
		zapLog = l
	}
}

// WithPrintField set print field
func WithPrintField(fields ...zap.Field) Option {
	return func(o *options) {
		o.zapFields = fields
	}
}

// WithAlarm enable alarm and notify, except windows
func WithAlarm(opts ...AlarmOption) Option {
	return func(o *options) {
		if runtime.GOOS == "windows" {
			return
		}
		ao := &alarmOptions{}
		ao.apply(opts...)
		o.enableAlarm = true
	}
}

// WithCustomHandler set custom handler and interval, will replace default print stat data handler
func WithCustomHandler(handler func(ctx context.Context, sd *StatData) error) Option {
	return func(o *options) {
		o.customHandler = handler
	}
}

// Init initialize statistical information
func Init(opts ...Option) {
	o := &options{}
	o.apply(opts...)

	//nolint
	go func() {
		printTick := time.NewTicker(printInfoInterval)
		defer printTick.Stop()
		sg := newStatGroup()

		for {
			select {
			case <-printTick.C:
				data := getStatData()
				if o.enableAlarm {
					handleAlarm(sg, data, o)
				}
				if o.customHandler == nil {
					printUsageInfo(data, o.zapFields...)
				} else {
					handleCustom(data, o)
				}
			}
		}

	}()
}

func handleAlarm(sg *statGroup, data *StatData, o *options) {
	if runtime.GOOS == "windows" { // Windows system does not support alarm
		return
	}
	if o.enableAlarm {
		if sg.check(data) {
			sendSystemSignForLinux()
		}
	}
}

func handleCustom(data *StatData, o *options) {
	ctx, _ := context.WithTimeout(context.Background(), printInfoInterval) //nolint
	defer func() { _ = recover() }()
	err := o.customHandler(ctx, data)
	if err != nil {
		zapLog.Warn("custom handler error", zap.Error(err))
	}
}

// nolint
func sendSystemSignForLinux() {
	select {
	case notifyCh <- struct{}{}:
	default:
	}
}

func getStatData() *StatData {
	defer func() { _ = recover() }()

	mSys := mem.GetSystemMemory()
	mProc := mem.GetProcessMemory()
	cSys := cpu.GetSystemCPU()
	cProc := cpu.GetProcess()

	var cors int32
	for _, ci := range cSys.CPUInfo {
		cors += ci.Cores
	}

	sys := System{
		CPUUsage: cSys.UsagePercent,
		CPUCores: cors,
		MemTotal: mSys.Total,
		MemFree:  mSys.Free,
		MemUsage: float64(int(math.Round(mSys.UsagePercent))), // rounding
	}
	proc := Process{
		CPUUsage:   cProc.UsagePercent,
		RSS:        cProc.RSS,
		VMS:        cProc.VMS,
		Alloc:      mProc.Alloc,
		TotalAlloc: mProc.TotalAlloc,
		Sys:        mProc.Sys,
		NumGc:      mProc.NumGc,
		Goroutines: runtime.NumGoroutine(),
	}

	return &StatData{
		Sys:  sys,
		Proc: proc,
	}
}

func printUsageInfo(statData *StatData, fields ...zap.Field) {
	fields = append(fields, zap.Any("system", statData.Sys), zap.Any("process", statData.Proc))
	zapLog.Info("statistics", fields...)
}

// System information
type System struct {
	CPUUsage float64 `json:"cpu_usage"` // system cpu usage, unit(%)
	MemUsage float64 `json:"mem_usage"` // system memory usage, unit(%)
	CPUCores int32   `json:"cpu_cores"` // cpu cores, multiple cpu accumulation
	MemTotal uint64  `json:"mem_total"` // system total physical memory, unit(M)
	MemFree  uint64  `json:"mem_free"`  // system free physical memory, unit(M)
}

// Process information
type Process struct {
	CPUUsage   float64 `json:"cpu_usage"`   // process cpu usage, unit(%)
	RSS        uint64  `json:"rss"`         // use of physical memory, unit(M)
	VMS        uint64  `json:"vms"`         // use of virtual memory, unit(M)
	Alloc      uint64  `json:"alloc"`       // allocated memory capacity, unit(M)
	TotalAlloc uint64  `json:"total_alloc"` // cumulative allocated memory capacity, unit(M)
	Sys        uint64  `json:"sys"`         // requesting memory capacity from the system, unit(M)
	NumGc      uint32  `json:"num_gc"`      // number of GC cycles
	Goroutines int     `json:"goroutines"`  // number of goroutines
}

// StatData statistical data
type StatData struct {
	Sys  System
	Proc Process
}
