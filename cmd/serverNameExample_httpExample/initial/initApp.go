// Package initial is the package that starts the service to initialize the service, including
// the initialization configuration, service configuration, connecting to the database, and
// resource release needed when shutting down the service.
package initial

import (
	"fmt"
	"os"
	"strconv"
	"time"

	ginAuth "github.com/Eric-Guo/sponge/pkg/gin/middleware/auth"

	"github.com/Eric-Guo/sponge/pkg/conf"
	"github.com/Eric-Guo/sponge/pkg/logger"
	"github.com/Eric-Guo/sponge/pkg/stat"
	"github.com/Eric-Guo/sponge/pkg/tracer"

	"github.com/Eric-Guo/sponge/configs"
	"github.com/Eric-Guo/sponge/internal/config"
	"github.com/Eric-Guo/sponge/internal/database"
)

// InitApp initial app configuration
func InitApp(configFile, version string) {
	initConfig(configFile, version)
	cfg := config.Get()

	// initializing log
	_, err := logger.Init(
		logger.WithLevel(cfg.Logger.Level),
		logger.WithFormat(cfg.Logger.Format),
		logger.WithSave(
			cfg.Logger.IsSave,
			//logger.WithFileName(cfg.Logger.LogFileConfig.Filename),
			//logger.WithFileMaxSize(cfg.Logger.LogFileConfig.MaxSize),
			//logger.WithFileMaxBackups(cfg.Logger.LogFileConfig.MaxBackups),
			//logger.WithFileMaxAge(cfg.Logger.LogFileConfig.MaxAge),
			//logger.WithFileIsCompression(cfg.Logger.LogFileConfig.IsCompression),
		),
	)
	if err != nil {
		panic(err)
	}
	logger.Debug(config.Show())
	logger.Info("[logger] was initialized")

	// initializing tracing
	if cfg.App.EnableTrace {
		tracer.InitWithConfig(
			cfg.App.Name,
			cfg.App.Env,
			cfg.App.Version,
			cfg.Jaeger.AgentHost,
			strconv.Itoa(cfg.Jaeger.AgentPort),
			cfg.App.TracingSamplingRate,
		)
		logger.Info("[tracer] was initialized")
	}

	// initializing the print system and process resources
	if cfg.App.EnableStat {
		stat.Init(
			stat.WithLog(logger.Get()),
			stat.WithAlarm(), // invalid if it is windows, the default threshold for cpu and memory is 0.8, you can modify them
			stat.WithPrintField(logger.String("service_name", cfg.App.Name), logger.String("host", cfg.App.Host)),
		)
		logger.Info("[resource statistics] was initialized")
	}

	// initializing database
	database.InitDB()
	logger.Infof("[%s] was initialized", cfg.Database.Driver)
	database.InitCache(cfg.App.CacheType)
	if cfg.App.CacheType != "" {
		logger.Infof("[%s] was initialized", cfg.App.CacheType)
	}
	if cfg.JWT.SigningKey != "" && cfg.JWT.SigningKey != "change-me" {
		ginAuth.InitAuth([]byte(cfg.JWT.SigningKey), time.Duration(cfg.JWT.Expire)*time.Second)
	}
}

func initConfig(configFile, version string) {
	getConfigFromLocal(configFile)

	if version != "" {
		config.Get().App.Version = version
	}
}

// get configuration from local configuration file
func getConfigFromLocal(configFile string) {
	if configFile == "" {
		configFile = configs.Location("serverNameExample.yml")
	}
	err := loadConfig(configFile)
	if err != nil {
		panic("init config error: " + err.Error())
	}
}

// Keep startup defaults and environment overrides outside the configuration
// structs rewritten by make update-config.
func loadConfig(configFile string) error {
	cfg := &config.Config{HTTP: config.HTTP{GzipJitter: 32}}
	if err := conf.Parse(configFile, cfg); err != nil {
		return err
	}
	if err := applyThrusterEnvironment(cfg); err != nil {
		return err
	}
	config.Set(cfg)
	return nil
}

// Thruster environment names remain compatible with existing deployments.
// Prefixed values take precedence, and environment values override YAML.
func applyThrusterEnvironment(cfg *config.Config) error {
	for _, setting := range []struct {
		name   string
		target *bool
	}{
		{"GZIP_COMPRESSION_ENABLED", &cfg.HTTP.GzipEnabled},
		{"GZIP_COMPRESSION_DISABLE_ON_AUTH", &cfg.HTTP.GzipDisableOnAuth},
		{"FORWARD_HEADERS", &cfg.Proxy.ForwardHeaders},
	} {
		if raw, ok := thrusterEnv(setting.name); ok {
			value, err := strconv.ParseBool(raw)
			if err != nil {
				return fmt.Errorf("%s: %w", setting.name, err)
			}
			*setting.target = value
		}
	}
	if raw, ok := thrusterEnv("GZIP_COMPRESSION_JITTER"); ok {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("GZIP_COMPRESSION_JITTER: %w", err)
		}
		cfg.HTTP.GzipJitter = value
	}
	return nil
}

func thrusterEnv(name string) (string, bool) {
	if value, ok := os.LookupEnv("THRUSTER_" + name); ok {
		return value, true
	}
	return os.LookupEnv(name)
}
