package utils

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

var Logger *logrus.Logger

// InitLogger initializes the global logger with structured logging
func InitLogger() {
	Logger = logrus.New()

	// Set output to stdout
	Logger.SetOutput(os.Stdout)

	// Set log level from environment or default to Info
	logLevel := os.Getenv("LOG_LEVEL")
	switch logLevel {
	case "debug":
		Logger.SetLevel(logrus.DebugLevel)
	case "info":
		Logger.SetLevel(logrus.InfoLevel)
	case "warn":
		Logger.SetLevel(logrus.WarnLevel)
	case "error":
		Logger.SetLevel(logrus.ErrorLevel)
	default:
		Logger.SetLevel(logrus.InfoLevel)
	}

	// Set formatter based on environment
	env := os.Getenv("ENV")
	if env == "production" {
		// JSON formatter for production (better for log aggregation)
		Logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	} else {
		// Text formatter for development (human-readable)
		Logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
		})
	}

	Logger.Info("Logger initialized successfully")
}

// LogInfo logs informational messages with structured fields
func LogInfo(message string, fields map[string]interface{}) {
	if Logger == nil {
		InitLogger()
	}
	Logger.WithFields(fields).Info(message)
}

// LogDebug logs debug messages with structured fields
func LogDebug(message string, fields map[string]interface{}) {
	if Logger == nil {
		InitLogger()
	}
	Logger.WithFields(fields).Debug(message)
}

// LogWarn logs warning messages with structured fields
func LogWarn(message string, fields map[string]interface{}) {
	if Logger == nil {
		InitLogger()
	}
	Logger.WithFields(fields).Warn(message)
}

// LogError logs error messages with structured fields
func LogError(message string, err error, fields map[string]interface{}) {
	if Logger == nil {
		InitLogger()
	}
	if fields == nil {
		fields = make(map[string]interface{})
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	Logger.WithFields(fields).Error(message)
}

// LogFatal logs fatal messages and exits
func LogFatal(message string, err error, fields map[string]interface{}) {
	if Logger == nil {
		InitLogger()
	}
	if fields == nil {
		fields = make(map[string]interface{})
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	Logger.WithFields(fields).Fatal(message)
}

// LogHTTPRequest logs HTTP request details
func LogHTTPRequest(method, path string, statusCode int, duration time.Duration, userID uint) {
	if Logger == nil {
		InitLogger()
	}
	Logger.WithFields(logrus.Fields{
		"method":      method,
		"path":        path,
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
		"user_id":     userID,
	}).Info("HTTP Request")
}

// LogWebSocketEvent logs WebSocket events
func LogWebSocketEvent(event string, userID uint, roomID uint, details map[string]interface{}) {
	if Logger == nil {
		InitLogger()
	}
	fields := logrus.Fields{
		"event":   event,
		"user_id": userID,
		"room_id": roomID,
	}
	for k, v := range details {
		fields[k] = v
	}
	Logger.WithFields(fields).Info("WebSocket Event")
}

// LogDatabaseQuery logs database query performance
func LogDatabaseQuery(query string, duration time.Duration, rowsAffected int64) {
	if Logger == nil {
		InitLogger()
	}
	Logger.WithFields(logrus.Fields{
		"query":         query,
		"duration_ms":   duration.Milliseconds(),
		"rows_affected": rowsAffected,
	}).Debug("Database Query")
}

// LogAuthEvent logs authentication events
func LogAuthEvent(event string, userID uint, email string, success bool, details map[string]interface{}) {
	if Logger == nil {
		InitLogger()
	}
	fields := logrus.Fields{
		"event":   event,
		"user_id": userID,
		"email":   email,
		"success": success,
	}
	for k, v := range details {
		fields[k] = v
	}
	Logger.WithFields(fields).Info("Auth Event")
}

// LogPanic logs panic recovery details
func LogPanic(err interface{}, stackTrace []byte, requestPath string, userID uint) {
	if Logger == nil {
		InitLogger()
	}
	Logger.WithFields(logrus.Fields{
		"panic":       err,
		"stack_trace": string(stackTrace),
		"path":        requestPath,
		"user_id":     userID,
	}).Error("Panic Recovered")
}
