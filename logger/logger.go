package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

// LogLevel represents the severity level of a log entry
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Component string                 `json:"component,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
}

// Logger provides structured logging functionality
type Logger struct {
	component string
	level     LogLevel
	logger    *log.Logger
}

// NewLogger creates a new logger instance
func NewLogger(component string) *Logger {
	return &Logger{
		component: component,
		level:     getLogLevelFromEnv(),
		logger:    log.New(os.Stdout, "", 0),
	}
}

// WithRequestID returns a new logger with a request ID
func (l *Logger) WithRequestID(requestID string) *Logger {
	newLogger := *l
	newLogger.component = l.component + "[" + requestID + "]"
	return &newLogger
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields ...map[string]interface{}) {
	l.log(DEBUG, message, "", fields...)
}

// Info logs an info message
func (l *Logger) Info(message string, fields ...map[string]interface{}) {
	l.log(INFO, message, "", fields...)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields ...map[string]interface{}) {
	l.log(WARN, message, "", fields...)
}

// Error logs an error message
func (l *Logger) Error(message string, err error, fields ...map[string]interface{}) {
	errorStr := ""
	if err != nil {
		errorStr = err.Error()
	}
	l.log(ERROR, message, errorStr, fields...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(message string, err error, fields ...map[string]interface{}) {
	errorStr := ""
	if err != nil {
		errorStr = err.Error()
	}
	l.log(FATAL, message, errorStr, fields...)
	os.Exit(1)
}

// log performs the actual logging
func (l *Logger) log(level LogLevel, message, errorStr string, fields ...map[string]interface{}) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level.String(),
		Message:   message,
		Component: l.component,
		Error:     errorStr,
		Caller:    getCaller(),
	}

	// Merge all field maps
	if len(fields) > 0 {
		entry.Fields = make(map[string]interface{})
		for _, fieldMap := range fields {
			for k, v := range fieldMap {
				entry.Fields[k] = v
			}
		}
	}

	// Output as JSON
	if jsonData, err := json.Marshal(entry); err == nil {
		l.logger.Println(string(jsonData))
	} else {
		// Fallback to simple logging if JSON marshaling fails
		l.logger.Printf("[%s] %s: %s - %s", level.String(), l.component, message, errorStr)
	}
}

// getCaller returns the caller information
func getCaller() string {
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		return ""
	}
	
	// Get just the filename, not the full path
	parts := strings.Split(file, "/")
	if len(parts) > 0 {
		file = parts[len(parts)-1]
	}
	
	return fmt.Sprintf("%s:%d", file, line)
}

// getLogLevelFromEnv gets the log level from environment variable
func getLogLevelFromEnv() LogLevel {
	levelStr := strings.ToUpper(os.Getenv("LOG_LEVEL"))
	switch levelStr {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO // Default to INFO level
	}
}

// Global logger instances for different components
var (
	MainLogger       = NewLogger("main")
	ConfigLogger     = NewLogger("config")
	ClientLogger     = NewLogger("client")
	MiddlewareLogger = NewLogger("middleware")
	ModelsLogger     = NewLogger("models")
)