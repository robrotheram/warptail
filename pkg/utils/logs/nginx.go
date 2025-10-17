package logs

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"time"
)

// Format logs in the format used by NGINX
type HttpResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
	StatusCode  int
	Size        int
}

func (lrw *HttpResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := lrw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not support hijacking")
}

func (lrw *HttpResponseWriter) WriteHeader(code int) {
	if !lrw.wroteHeader {
		lrw.StatusCode = code
		lrw.ResponseWriter.WriteHeader(code)
		lrw.wroteHeader = true
	}
}

func (lrw *HttpResponseWriter) Write(b []byte) (int, error) {
	if !lrw.wroteHeader {
		// Default to 200 if WriteHeader not called yet
		lrw.WriteHeader(http.StatusOK)
	}
	n, err := lrw.ResponseWriter.Write(b)
	lrw.Size += n
	return n, err
}

type LoggingResponseWriter struct {
	accessLog *DualWriter
	errorLog  *DualWriter
}

func NewAccessLogWriter(path string) (*LoggingResponseWriter, error) {
	accessFilePath := filepath.Join(path, "access.log")
	errorFilePath := filepath.Join(path, "error.log")

	accessFile, err := NewDualWriter(accessFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open log access log file: %w", err)
	}

	errorFile, err := NewDualWriter(errorFilePath)
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

		// Only log if this is a proxy call (set by proxy middleware)
		if isProxy, ok := r.Context().Value("isProxy").(bool); ok && isProxy {
			lrw.LogRequest(r, start, hrw.StatusCode, hrw.Size)
		}
	})
}

func getClientIP(r *http.Request) string {
	// Check if the request is behind a proxy
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
}

func (lrw *LoggingResponseWriter) LogRequest(r *http.Request, start time.Time, statusCode int, size int) {
	logLine := fmt.Sprintf(
		`%s - - [%s] "%s %s %s" %d %d "%s" "%s"`,
		getClientIP(r),
		start.Format("02/Jan/2006:15:04:05"),
		r.Method,
		r.RequestURI,
		r.Proto,
		statusCode,
		size,
		r.Referer(),
		r.UserAgent(),
	)
	if statusCode >= 500 {
		lrw.LogError(r, fmt.Errorf("server error: %d", statusCode))
	}
	lrw.accessLog.WriteString(logLine + "\n")
}

func (lrw *LoggingResponseWriter) GetLogs(logType string) ([]string, error) {
	var logFile *DualWriter
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

func (lrw *LoggingResponseWriter) LogError(r *http.Request, err error) {
	timestamp := time.Now().Format("2006/01/02 15:04:05")
	logLine := fmt.Sprintf("[%s] [error] client: %s, request: %s %s, error: %v", timestamp, getClientIP(r), r.Method, r.RequestURI, err)
	lrw.errorLog.WriteString(logLine + "\n")
}

func (lrw *LoggingResponseWriter) Close() error {
	if err := lrw.accessLog.Close(); err != nil {
		return fmt.Errorf("error closing access log file: %v", err)
	}
	if err := lrw.errorLog.Close(); err != nil {
		return fmt.Errorf("error closing error log file: %v", err)
	}
	return nil
}
