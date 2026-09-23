## stat

Statistics on system and process cpu and memory information, alarm notification and custom handler support.

<br>

### Example of use

```go
    import "github.com/Eric-Guo/sponge/pkg/stat"

    l, _ := zap.NewDevelopment()
    stat.Init(
        stat.WithLog(l),
        stat.WithPrintInterval(time.Minute),
        stat.WithPrintField(logger.String("service_name", cfg.App.Name), logger.String("host", cfg.App.Host)), // add custom fields to log
        stat.WithEnableAlarm(stat.WithCPUThreshold(0.85), stat.WithMemoryThreshold(0.85)), // enable alarm and trigger collect profile data, invalid if it is windows
        //stat.WithCustomHandler(func(ctx context.Context, sd *stat.StatData) error { // it will be replace default print handler
        //    //push stat data to remote server (prometheus, influxdb, etc.) or do something else
        //    return nil
        //}),
    )
```
