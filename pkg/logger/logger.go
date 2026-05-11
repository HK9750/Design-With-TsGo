package logger

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

var levelNames = map[Level]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
}

var levelColors = map[Level]string{
	DEBUG: "\033[36m",
	INFO:  "\033[32m",
	WARN:  "\033[33m",
	ERROR: "\033[31m",
}

const colorReset = "\033[0m"

type Logger struct {
	mu       sync.Mutex
	out      io.Writer
	level    Level
	useColor bool
	fields   map[string]any
}

var defaultLogger = &Logger{
	out:      os.Stdout,
	level:    DEBUG,
	useColor: true,
	fields:   make(map[string]any),
}

func New(out io.Writer, level Level, useColor bool) *Logger {
	return &Logger{
		out:      out,
		level:    level,
		useColor: useColor,
		fields:   make(map[string]any),
	}
}

func SetLevel(level Level) { defaultLogger.level = level }

func SetOutput(out io.Writer) { defaultLogger.out = out }

func SetColor(enabled bool) { defaultLogger.useColor = enabled }

func (l *Logger) WithField(key string, value any) *Logger {
	fields := make(map[string]any, len(l.fields)+1)
	for k, v := range l.fields {
		fields[k] = v
	}
	fields[key] = value
	return &Logger{out: l.out, level: l.level, useColor: l.useColor, fields: fields}
}

func (l *Logger) log(level Level, format string, args ...any) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().Format("15:04:05.000")
	levelStr := levelNames[level]

	var caller string
	if level >= DEBUG {
		_, file, line, ok := runtime.Caller(2)
		if ok {
			if idx := strings.LastIndexByte(file, '/'); idx >= 0 {
				file = file[idx+1:]
			}
			caller = fmt.Sprintf("%s:%d", file, line)
		}
	}

	msg := fmt.Sprintf(format, args...)
	fieldStr := ""
	if len(l.fields) > 0 {
		parts := make([]string, 0, len(l.fields))
		for k, v := range l.fields {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
		fieldStr = " [" + strings.Join(parts, " ") + "]"
	}

	if l.useColor {
		color := levelColors[level]
		fmt.Fprintf(l.out, "%s%s | %s%-5s%s | %s%s | %s\n",
			now, fieldStr, color, levelStr, colorReset, msg, caller,
		)
	} else {
		fmt.Fprintf(l.out, "%s%s | %-5s | %s | %s\n",
			now, fieldStr, levelStr, msg, caller,
		)
	}
}

func (l *Logger) Debug(format string, args ...any) { l.log(DEBUG, format, args...) }
func (l *Logger) Info(format string, args ...any)  { l.log(INFO, format, args...) }
func (l *Logger) Warn(format string, args ...any)  { l.log(WARN, format, args...) }
func (l *Logger) Error(format string, args ...any) { l.log(ERROR, format, args...) }

// Operation creates a logger with an operation field and logs the start, returning a done function.
func (l *Logger) Operation(name string, format string, args ...any) func() {
	opLogger := l.WithField("op", name)
	opLogger.Info("START: "+format, args...)
	start := time.Now()
	return func() {
		elapsed := time.Since(start)
		opLogger.WithField("elapsed", elapsed.String()).Info("DONE: " + name)
	}
}

// Step logs an intermediate step within an operation
func (l *Logger) Step(format string, args ...any) {
	l.Debug("  -> " + format, args...)
}

// Package-level convenience functions
func Debug(format string, args ...any) { defaultLogger.Debug(format, args...) }
func Info(format string, args ...any)  { defaultLogger.Info(format, args...) }
func Warn(format string, args ...any)  { defaultLogger.Warn(format, args...) }
func Error(format string, args ...any) { defaultLogger.Error(format, args...) }

func Operation(name string, format string, args ...any) func() {
	return defaultLogger.Operation(name, format, args...)
}

func Step(format string, args ...any) { defaultLogger.Step(format, args...) }

func WithField(key string, value any) *Logger { return defaultLogger.WithField(key, value) }

// Section logs a visually distinct section header
func Section(title string) {
	fmt.Fprintf(defaultLogger.out, "\n%s━━━ %s ━━━%s\n", "\033[1;35m", title, colorReset)
}

// KeyValue logs a key=value pair at info level
func KeyValue(key string, value any) {
	Info("%s = %v", key, value)
}
