package utils

import (
	"bytes"
	"log"
	"os"
	"strings"
	"warptail/pkg/utils/logs"

	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var Logger logr.Logger
var LogBuffer *bytes.Buffer
var RequestLogger *logs.LoggingResponseWriter

const DefaultLogOutput = "stdout"
const DefaultLevel = "info"
const DefaultLogPath = "/var/log/warptail"

type LoggingConfig struct {
	Format string `yaml:"format"`
	Level  string `yaml:"level"`
	Output string `yaml:"output"`
	Path   string `yaml:"path"`
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
	if len(cfg.Path) == 0 {
		cfg.Path = DefaultLogPath
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

	// Initialize the in-memory log buffer
	LogBuffer = &bytes.Buffer{}

	var writers []zapcore.WriteSyncer
	writers = append(writers, zapcore.AddSync(LogBuffer)) // Add in-memory buffer

	if logCfg.Output != DefaultLogOutput {
		file, err := os.OpenFile(logCfg.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Failed to open log file %s: %v", logCfg.Output, err)
		}
		writers = append(writers, zapcore.AddSync(file))
	} else {
		writers = append(writers, zapcore.AddSync(os.Stdout))
	}

	opts.DestWriter = zapcore.NewMultiWriteSyncer(writers...)

	Logger = zap.New(zap.UseFlagOptions(&opts))

	err := os.MkdirAll(logCfg.Path, os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to open log file %s: %v", logCfg.Output, err)
	}
	RequestLogger, _ = logs.NewAccessLogWriter(logCfg.Path)
}

// GetLogs returns the logs stored in memory as a string
func GetLogs() []string {
	return strings.Split(LogBuffer.String(), "\n")
}
