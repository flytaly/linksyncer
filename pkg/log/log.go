package log

import (
	"fmt"
	"log"
	"os"
	"time"
)

type Level int

const (
	Info Level = iota
	Warning
	Error
)

type Record struct {
	Level Level
	Ts    time.Time
	Msg   string
}

type Logger interface {
	Error(format string, v ...any)
	Warning(format string, v ...any)
	Info(format string, v ...any)
	Close() error
}

func New(path string, channel chan Record) Logger {
	if path != "" {
		file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			log.Fatalf("Unable to open log file. %s", err)
			return nil
		}
		return &StdLog{
			err:     log.New(file, "ERROR ", log.Ldate|log.Ltime),
			wrn:     log.New(file, "WARN ", log.Ldate|log.Ltime),
			inf:     log.New(file, "INFO ", log.Ldate|log.Ltime),
			file:    file,
			channel: channel,
		}
	}
	return &StdLog{
		// err:     log.New(os.Stderr, "", 0),
		// wrn:     log.New(os.Stderr, "", 0),
		// inf:     log.New(os.Stdout, "", 0),
		channel: channel,
	}
}

type StdLog struct {
	err, wrn, inf *log.Logger
	file          *os.File
	channel       chan Record
}

func (l *StdLog) Error(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	l.Send(Record{Error, time.Now(), msg})
	if l.err != nil {
		_ = l.err.Output(2, fmt.Sprintf(format, v...))
	}
}

func (l *StdLog) Info(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	l.Send(Record{Info, time.Now(), msg})
	if l.inf != nil {
		_ = l.inf.Output(2, msg)
	}
}

func (l *StdLog) Warning(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	l.Send(Record{Warning, time.Now(), msg})
	if l.wrn != nil {
		_ = l.wrn.Output(2, msg)
	}
}

func (l *StdLog) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *StdLog) Send(entry Record) {
	if l.channel != nil {
		go func() {
			l.channel <- entry
		}()
	}
}

type EmptyLog struct{}

func NewEmptyLog() Logger { return EmptyLog{} }

func (l EmptyLog) Error(string, ...any)   {}
func (l EmptyLog) Warning(string, ...any) {}
func (l EmptyLog) Info(string, ...any)    {}
func (l EmptyLog) Close() error           { return nil }
