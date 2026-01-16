package utils

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"warptail/pkg/utils/logs"

	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	ctrl "sigs.k8s.io/controller-runtime"
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
		path := filepath.Join(logCfg.Path, "warptail.log")
		file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Failed to open log file %s: %v", path, err)
		}
		writers = append(writers, zapcore.AddSync(file))
	} else {
		writers = append(writers, zapcore.AddSync(os.Stdout))
	}

	opts.DestWriter = zapcore.NewMultiWriteSyncer(writers...)

	Logger = zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(Logger)

	err := os.MkdirAll(logCfg.Path, os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to open log file %s: %v", logCfg.Output, err)
	}
	RequestLogger, _ = logs.NewAccessLogWriter(logCfg.Path)
}

// GetLogs returns the logs stored in memory as a slice of single-line strings
func GetLogs() []string {
	raw := LogBuffer.String()
	lines := strings.Split(raw, "\n")
	var logs []string
	var entry strings.Builder

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		// If the line looks like a new log entry (starts with a timestamp), start a new entry
		if len(logs) == 0 || isNewLogEntry(line) {
			if entry.Len() > 0 {
				logs = append(logs, entry.String())
				entry.Reset()
			}
			entry.WriteString(strings.ReplaceAll(line, "\n", " "))
		} else {
			// Continuation of previous log (e.g., stack trace), append as space
			entry.WriteString(" " + strings.ReplaceAll(line, "\n", " "))
		}
	}
	if entry.Len() > 0 {
		logs = append(logs, entry.String())
	}
	return logs
}

// isNewLogEntry tries to detect if a line is a new log entry by checking for a timestamp at the start
func isNewLogEntry(line string) bool {
	// Example: 2024-05-31T12:34:56.789Z
	if len(line) > 20 && line[4] == '-' && line[7] == '-' && (line[10] == 'T' || line[10] == ' ') {
		return true
	}
	return false
}

func LogHttpError(r *http.Request, err error) {
	RequestLogger.LogError(r, err)
}
