package logger

import (
	"log"
	"os"
)

type Logger interface {
	// Debug(...any)
	// Debugf(template string, args ...any)

	Info(...any)
	Infof(template string, args ...any)

	// Warn(...any)
	// Warnf(template string, args ...any)

	// Error(...any)
	// Errorf(template string, args ...any)

	Fatal(...any)
	Fatalf(template string, args ...any)

	// Panic(...any)
	// Panicf(template string, args ...any)
}

type logger struct {
	logger *log.Logger
}

func New() Logger {
	l := &logger{
		log.New(os.Stdout, "", 0),
	}

	return l
}

func (l *logger) Info(args ...any) {
	l.logger.Print(args...)
}

func (l *logger) Infof(template string, args ...any) {
	l.logger.Printf(template, args...)
}

func (l *logger) Fatal(args ...any) {
	l.logger.Fatal(args...)
}

func (l *logger) Fatalf(template string, args ...any) {
	l.logger.Fatalf(template, args...)
}
