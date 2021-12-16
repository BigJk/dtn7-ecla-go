package ecla

import "log"

type Logger interface {
	Log(v ...interface{})
	Logf(format string, v ...interface{})
}

type EmptyLogger struct {
}

func (e *EmptyLogger) Log(v ...interface{}) {
}

func (e *EmptyLogger) Logf(format string, v ...interface{}) {
}

type LogLogger struct {
}

func (e *LogLogger) Log(v ...interface{}) {
	log.Println(v...)
}

func (e *LogLogger) Logf(format string, v ...interface{}) {
	log.Printf(format, v...)
}
