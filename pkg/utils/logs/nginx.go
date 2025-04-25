package logs

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Format logs in the format used by NGINX
type HttpResponseWriter struct {
	http.ResponseWriter
	StatusCode int
	Size       int
}

func (lrw *HttpResponseWriter) WriteHeader(code int) {
	lrw.StatusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *HttpResponseWriter) Write(b []byte) (int, error) {
	n, err := lrw.ResponseWriter.Write(b)
	lrw.Size += n
	return n, err
}

type LoggingResponseWriter struct {
	accessLog *os.File
	errorLog  *os.File
}

func NewAccessLogWriter(path string) (*LoggingResponseWriter, error) {
	accessFilePath := filepath.Join(path, "access.log")
	errorFilePath := filepath.Join(path, "error.log")

	accessFile, err := os.OpenFile(accessFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log access log file: %w", err)
	}

	errorFile, err := os.OpenFile(errorFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log error log file: %w", err)
	}

	lrw := &LoggingResponseWriter{
		accessLog: accessFile,
		errorLog:  errorFile,
	}
	return lrw, nil
}

func (lrw *LoggingResponseWriter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hrw := &HttpResponseWriter{ResponseWriter: w, StatusCode: 200}
		start := time.Now()
		next.ServeHTTP(hrw, r)
		lrw.LogRequest(r, start, hrw.StatusCode, hrw.Size)
	})
}

func (lrw *LoggingResponseWriter) LogRequest(r *http.Request, start time.Time, statusCode int, size int) {
	logLine := fmt.Sprintf(
		`%s - - [%s] "%s %s %s" %d %d "%s" "%s"`,
		r.RemoteAddr,
		start.Format("02/Jan/2006:15:04:05 -0700"),
		r.Method,
		r.RequestURI,
		r.Proto,
		statusCode,
		size,
		r.Referer(),
		r.UserAgent(),
	)
	lrw.accessLog.WriteString(logLine + "\n")
	lrw.accessLog.Sync()
}

func (lrw *LoggingResponseWriter) GetLogs(logType string) ([]string, error) {
	var logFile *os.File
	switch logType {
	case "access":
		logFile = lrw.accessLog
	case "error":
		logFile = lrw.errorLog
	default:
		return nil, fmt.Errorf("invalid log type: %s", logType)
	}
	var logs []string
	scanner := bufio.NewScanner(logFile)
	for scanner.Scan() {
		logs = append(logs, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return logs, nil
}

func (lrw *LoggingResponseWriter) Close() {
	if err := lrw.accessLog.Close(); err != nil {
		log.Printf("Error closing access log file: %v", err)
	}
	if err := lrw.errorLog.Close(); err != nil {
		log.Printf("Error closing error log file: %v", err)
	}
}
