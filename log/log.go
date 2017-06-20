package log

import (
	"log"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New initializes a new Zap logger with namespaced keys.
func New() (*zap.Logger, error) {
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.EncoderConfig = zapcore.EncoderConfig{
		MessageKey:     "log.msg",
		LevelKey:       "log.level",
		TimeKey:        "log.ts",
		NameKey:        "log.name",
		CallerKey:      "log.source",
		StacktraceKey:  "log.stack",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.EpochTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Setup the logger with the base context.
	logger, err := loggerConfig.Build()
	if err != nil {
		return logger, err
	}

	// Get server metadata from environment variables.
	serverPublicHostname := os.Getenv("SERVER_PUBLIC_HOSTNAME")
	serverPublicIpv4 := os.Getenv("SERVER_PUBLIC_IPV4")
	serverLocalHostname := os.Getenv("SERVER_LOCAL_HOSTNAME")
	serverLocalIpv4 := os.Getenv("SERVER_LOCAL_IPV4")
	serverNickname := os.Getenv("SERVER_NICKNAME")

	// Augment with log format and server metadata.
	logger = logger.With(
		zap.String("log.format", "json"),
		zap.String("server.public.hostname", serverPublicHostname),
		zap.String("server.public.ipv4", serverPublicIpv4),
		zap.String("server.local.hostname", serverLocalHostname),
		zap.String("server.local.ipv4", serverLocalIpv4),
		zap.String("server.nickname", serverNickname),
	)

	return logger, nil
}

// Fatal calls Fatal on the stdlib logger.
func Fatal(v ...interface{}) {
	log.Fatal(v)
}

// Fatalf calls Fatalf on the stdlib logger.
func Fatalf(f string, v ...interface{}) {
	log.Fatalf(f, v...)
}

// Fatalln calls Fatalln on the stdlib logger.
func Fatalln(v ...interface{}) {
	log.Fatalln(v...)
}
