// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package log

import (
	"fmt"
	"log"
	"log/syslog"
	"os"
	"strings"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	debug     *log.Logger
	info      *log.Logger
	notice    *log.Logger
	warning   *log.Logger
	err       *log.Logger
	crit      *log.Logger
	verbosity syslog.Priority
	writer    *syslog.Writer
)

// Parse a log level from a string and return an integer value
func ParseLogLevel(s string) (syslog.Priority, error) {
	var prio syslog.Priority
	switch strings.ToLower(s) {
	case "debug":
		prio = syslog.LOG_DEBUG
	case "info":
		prio = syslog.LOG_INFO
	case "notice":
		prio = syslog.LOG_NOTICE
	case "warning", "warn":
		prio = syslog.LOG_WARNING
	case "err", "error":
		prio = syslog.LOG_ERR
	case "crit", "critical":
		prio = syslog.LOG_CRIT
	default:
		return 0, fmt.Errorf("invalid log level %q", s)
	}
	return prio, nil
}

// Initialize the logging system.
// Redirect messages to syslog if run by systemd.
func InitLogging(level syslog.Priority) error {
	verbosity = level
	if os.Getenv("INVOCATION_ID") != "" {
		// executed by systemd
		w, err := syslog.New(syslog.LOG_DAEMON, "")
		if err != nil {
			return err
		}
		writer = w
	} else {
		flags := log.Ltime | log.Lshortfile
		debug = log.New(os.Stderr, "DEBUG   ", flags)
		info = log.New(os.Stderr, "INFO    ", flags)
		notice = log.New(os.Stderr, "NOTICE  ", flags)
		warning = log.New(os.Stderr, "WARNING ", flags)
		err = log.New(os.Stderr, "ERROR   ", flags)
		crit = log.New(os.Stderr, "CRIT    ", flags)
	}
	return nil
}

func format(message string, args ...any) string {
	return fmt.Sprintf(strings.TrimSpace(message)+"\n", args...)
}

// Write a DEBUG message to the log
func Debugf(message string, args ...any) {
	if verbosity < syslog.LOG_DEBUG {
		return
	}
	msg := format(message, args...)
	if writer != nil {
		_ = writer.Debug(msg)
	} else if debug != nil {
		_ = debug.Output(2, msg)
	}
}

// Write an INFO message to the log
func Infof(message string, args ...any) {
	if verbosity < syslog.LOG_INFO {
		return
	}
	msg := format(message, args...)
	if writer != nil {
		_ = writer.Info(msg)
	} else if info != nil {
		_ = info.Output(2, msg)
	}
}

// Write a NOTICE message to the log
func Noticef(message string, args ...any) {
	if verbosity < syslog.LOG_NOTICE {
		return
	}
	msg := format(message, args...)
	if writer != nil {
		_ = writer.Notice(msg)
	} else if notice != nil {
		_ = notice.Output(2, msg)
	}
}

// Write a WARNING message to the log
func Warningf(message string, args ...any) {
	if verbosity < syslog.LOG_WARNING {
		return
	}
	msg := format(message, args...)
	if writer != nil {
		_ = writer.Warning(msg)
	} else if warning != nil {
		_ = warning.Output(2, msg)
	}
}

// Write an ERR message to the log
func Errf(message string, args ...any) {
	if verbosity < syslog.LOG_ERR {
		return
	}
	msg := format(message, args...)
	if writer != nil {
		_ = writer.Err(msg)
	} else if err != nil {
		_ = err.Output(2, msg)
	}
}

func ErrorLogger() *log.Logger {
	return err
}

// Write a CRIT message to the log
func Critf(message string, args ...any) {
	if verbosity < syslog.LOG_CRIT {
		return
	}
	msg := format(message, args...)
	if writer != nil {
		_ = writer.Crit(msg)
	} else if crit != nil {
		_ = crit.Output(2, msg)
	}
}

type sink struct {
	calldepth int
}

func (s *sink) Enabled(level int) bool {
	if level < 0 {
		level = 0
	}
	switch level {
	case 0:
		return verbosity >= syslog.LOG_NOTICE
	case 1:
		return verbosity >= syslog.LOG_INFO
	default:
		return verbosity >= syslog.LOG_DEBUG
	}
}

func (s *sink) Error(e error, msg string, args ...any) {
	if verbosity < syslog.LOG_ERR {
		return
	}
	message := format("%s: %v", msg, args)
	if writer != nil {
		_ = writer.Err(message)
	} else {
		_ = err.Output(s.calldepth, message)
	}
}

func (s *sink) Info(level int, msg string, args ...any) {
	if level < 0 {
		level = 0
	}
	message := format("%s: %v", msg, args)
	switch level {
	case 0:
		if writer != nil {
			_ = writer.Err(message)
		} else {
			_ = err.Output(s.calldepth, message)
		}
	case 1:
		if writer != nil {
			_ = writer.Info(message)
		} else {
			_ = info.Output(s.calldepth, message)
		}
	default:
		if writer != nil {
			_ = writer.Debug(message)
		} else {
			_ = debug.Output(s.calldepth, message)
		}
	}
}

func (s *sink) Init(info logr.RuntimeInfo) {
	s.calldepth = info.CallDepth + 2
}

func (s *sink) WithValues(keysAndValues ...any) logr.LogSink {
	return s
}

func (s *sink) WithName(name string) logr.LogSink {
	return s
}

func OvsdbLogger() *logr.Logger {
	l := logr.New(new(sink))
	return &l
}

// used by prometheus error logging
func (s *sink) Println(args ...any) {
	msg := fmt.Sprintln(args...)
	if writer != nil {
		_ = writer.Err(msg)
	} else {
		_ = err.Output(2, msg)
	}
}

func PrometheusLogger() promhttp.Logger {
	return new(sink)
}
