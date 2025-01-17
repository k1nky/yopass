package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	yauth "github.com/k1nky/yopass/pkg/auth"
	"github.com/k1nky/yopass/pkg/server"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logLevel zapcore.Level

func init() {
	pflag.String("address", "", "listen address (default 0.0.0.0)")
	pflag.Int("port", 1337, "listen port")
	pflag.String("database", "memcached", "database backend ('memcached' or 'redis')")
	pflag.Int("max-length", 10000, "max length of encrypted secret")
	pflag.String("memcached", "localhost:11211", "memcached address")
	pflag.Int("metrics-port", -1, "metrics server listen port")
	pflag.String("redis", "redis://localhost:6379/0", "Redis URL")
	pflag.String("tls-cert", "", "path to TLS certificate")
	pflag.String("tls-key", "", "path to TLS key")
	pflag.String("auth-type", "no-auth", "auth type ('no-auth' or 'jwt')")
	pflag.String("auth-config", "auth.yaml", "jwt config location")
	pflag.Bool("force-onetime-secrets", false, "reject non onetime secrets from being created")
	pflag.CommandLine.AddGoFlag(&flag.Flag{Name: "log-level", Usage: "Log level", Value: &logLevel})

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	_ = viper.BindPFlags(pflag.CommandLine)

	pflag.Parse()
}

func main() {
	var err error

	logger := configureZapLogger()

	var db server.Database
	switch database := viper.GetString("database"); database {
	case "memcached":
		memcached := viper.GetString("memcached")
		db = server.NewMemcached(memcached)
		logger.Debug("configured Memcached", zap.String("address", memcached))
	case "redis":
		redis := viper.GetString("redis")
		db, err = server.NewRedis(redis)
		if err != nil {
			logger.Fatal("invalid Redis URL", zap.Error(err))
		}
		logger.Debug("configured Redis", zap.String("url", redis))
	default:
		logger.Fatal("unsupported database, expected 'memcached' or 'redis'", zap.String("database", database))
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	registry.MustRegister(prometheus.NewGoCollector())

	cert := viper.GetString("tls-cert")
	key := viper.GetString("tls-key")
	errc := make(chan error)

	var auth yauth.Auth
	switch authType := viper.GetString("auth-type"); authType {
	case "no-auth":
		auth = &yauth.NoAuth{}
	case "jwt":
		jwtAuth := &yauth.JwtAuth{}
		authConfig := viper.GetString("auth-config")
		logger.Info("Trying to load an auth config")
		err = jwtAuth.Load(authConfig)
		if err != nil {
			logger.Warn("failed to load, trying to create a new config", zap.String("error", err.Error()))
			jwtAuth = yauth.NewJwtAuth(authConfig, "yopass")
			err = jwtAuth.Save()
			if err != nil {
				logger.Fatal("failed to create", zap.String("error", err.Error()))
			}
			logger.Warn("--------------------------------")
			logger.Warn("Preset admin", zap.String("name", jwtAuth.Options.Users[0].Username), zap.String("password", jwtAuth.Options.Users[0].Password))
			logger.Warn("Please change that password")
			logger.Warn("--------------------------------")
			auth = jwtAuth
		}
	}

	go func() {
		addr := fmt.Sprintf("%s:%d", viper.GetString("address"), viper.GetInt("port"))
		logger.Info("Starting yopass server", zap.String("address", addr))
		y := server.New(db, viper.GetInt("max-length"), registry, viper.GetBool("force-onetime-secrets"), logger, auth)
		errc <- listenAndServe(addr, y.HTTPHandler(), cert, key)
	}()

	if port := viper.GetInt("metrics-port"); port > 0 {
		go func() {
			addr := fmt.Sprintf("%s:%d", viper.GetString("address"), port)
			logger.Info("Starting yopass metrics server", zap.String("address", addr))
			errc <- listenAndServe(addr, metricsHandler(registry), cert, key)
		}()
	}

	err = <-errc
	logger.Fatal("yopass stopped unexpectedly", zap.Error(err))
}

// listenAndServe starts a HTTP server on the given addr. It uses TLS if both
// certFile and keyFile are not empty.
func listenAndServe(addr string, h http.Handler, certFile, keyFile string) error {
	srv := &http.Server{
		Addr:      addr,
		Handler:   h,
		TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	}
	if certFile == "" || keyFile == "" {
		return srv.ListenAndServe()
	}
	return srv.ListenAndServeTLS(certFile, keyFile)
}

// metricsHandler builds a handler to serve Prometheus metrics
func metricsHandler(r *prometheus.Registry) http.Handler {
	mx := http.NewServeMux()
	mx.Handle("/metrics", promhttp.HandlerFor(r, promhttp.HandlerOpts{EnableOpenMetrics: true}))
	return mx
}

// configureZapLogger uses the `log-level` command line argument to set and replace the zap global logger.
func configureZapLogger() *zap.Logger {
	loggerCfg := zap.NewProductionConfig()
	loggerCfg.Level.SetLevel(logLevel)

	logger, err := loggerCfg.Build()
	if err != nil {
		log.Fatalf("Unable to build logger %v", err)
	}
	zap.ReplaceGlobals(logger)
	return logger
}
