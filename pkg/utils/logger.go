package utils

import (
	"log"
	"os"

	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var Logger logr.Logger

const DefaultLogOutput = "stdout"
const DefaultLevel = "info"

type LoggingConfig struct {
	Format string `yaml:"format"`
	Level  string `yaml:"level"`
	Output string `yaml:"output"`
}

func (cfg *LoggingConfig) Default() {
	if len(cfg.Format) == 0 {
		cfg.Format = DefaultLogOutput
	}
	if len(cfg.Output) == 0 {
		cfg.Output = DefaultLogOutput
	}
	if len(cfg.Level) == 0 {
		cfg.Level = DefaultLevel
	}
}

func setupLogger(logCfg LoggingConfig) {

	logCfg.Default()
	// Create an encoder config with custom colors
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var level = zapcore.InfoLevel
	switch logCfg.Level {
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	}

	opts := zap.Options{
		Development: false, // Production mode
		Level:       level,
	}

	if logCfg.Format == "json" {
		encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		opts.Encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		opts.Encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	if logCfg.Output != DefaultLogOutput {
		file, err := os.OpenFile(logCfg.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Failed to open log file %s: %v", logCfg.Output, err)
		}
		opts.DestWriter = file
	}
	Logger = zap.New(zap.UseFlagOptions(&opts))

}
