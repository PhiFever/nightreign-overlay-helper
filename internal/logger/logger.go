package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/PhiFever/nightreign-overlay-helper/pkg/utils"
)

// Level 表示日志级别
type Level int

const (
	DEBUG Level = iota
	INFO
	WARNING
	ERROR
	CRITICAL
)

var levelNames = map[Level]string{
	DEBUG:    "DEBUG",
	INFO:     "INFO",
	WARNING:  "WARNING",
	ERROR:    "ERROR",
	CRITICAL: "CRITICAL",
}

// Logger 是主日志记录器结构
type Logger struct{
	level   Level
	writers []io.Writer
	mu      sync.Mutex
}

var (
	globalLogger *Logger
	loggerMu     sync.Mutex
)

// Setup 使用指定的级别初始化全局日志记录器
func Setup(level Level) (*Logger, error) {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	if globalLogger != nil {
		return globalLogger, nil
	}

	logger := &Logger{
		level:   level,
		writers: []io.Writer{os.Stdout},
	}

	// 创建日志目录
	logDir, err := utils.GetAppDataPath("logs")
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(logDir), 0755); err != nil {
		return nil, err
	}

	// 创建日志文件
	date := time.Now().Format("2006-01-02")
	logFile := filepath.Join(filepath.Dir(logDir), "logs", date+".log")

	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	logger.writers = append(logger.writers, file)
	globalLogger = logger

	return logger, nil
}

// SetLevel 设置日志记录级别
func SetLevel(level Level) {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	if globalLogger == nil {
		globalLogger, _ = Setup(level)
	} else {
		globalLogger.level = level
	}
}

// GetLogger 返回全局日志记录器
func GetLogger() *Logger {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	if globalLogger == nil {
		globalLogger, _ = Setup(INFO)
	}

	return globalLogger
}

// log 使用指定的级别写入日志消息
func (l *Logger) log(level Level, msg string, includeTrace bool) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelName := levelNames[level]

	logMsg := fmt.Sprintf("%s [%s] %s\n", timestamp, levelName, msg)

	for _, w := range l.writers {
		w.Write([]byte(logMsg))
	}

	if includeTrace && level >= ERROR {
		trace := getStackTrace()
		traceMsg := fmt.Sprintf("%s [%s] %s\n", timestamp, levelName, trace)
		for _, w := range l.writers {
			w.Write([]byte(traceMsg))
		}
	}
}

// Debug 记录调试消息
func Debug(msg string) {
	GetLogger().log(DEBUG, msg, false)
}

// Debugf 记录格式化的调试消息
func Debugf(format string, args ...interface{}) {
	GetLogger().log(DEBUG, fmt.Sprintf(format, args...), false)
}

// Info 记录信息消息
func Info(msg string) {
	GetLogger().log(INFO, msg, false)
}

// Infof 记录格式化的信息消息
func Infof(format string, args ...interface{}) {
	GetLogger().log(INFO, fmt.Sprintf(format, args...), false)
}

// Warning 记录警告消息
func Warning(msg string) {
	GetLogger().log(WARNING, msg, false)
}

// Warningf 记录格式化的警告消息
func Warningf(format string, args ...interface{}) {
	GetLogger().log(WARNING, fmt.Sprintf(format, args...), false)
}

// Error 记录带有堆栈跟踪的错误消息
func Error(msg string) {
	GetLogger().log(ERROR, msg, true)
}

// Errorf 记录格式化的带有堆栈跟踪的错误消息
func Errorf(format string, args ...interface{}) {
	GetLogger().log(ERROR, fmt.Sprintf(format, args...), true)
}

// ErrorNoTrace 记录不带堆栈跟踪的错误消息
func ErrorNoTrace(msg string) {
	GetLogger().log(ERROR, msg, false)
}

// Critical 记录带有堆栈跟踪的严重消息
func Critical(msg string) {
	GetLogger().log(CRITICAL, msg, true)
}

// Criticalf 记录格式化的带有堆栈跟踪的严重消息
func Criticalf(format string, args ...interface{}) {
	GetLogger().log(CRITICAL, fmt.Sprintf(format, args...), true)
}

// getStackTrace 返回当前的堆栈跟踪
func getStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}
