package log

import (
	"fmt"
	"log"
	"os"
)

type Logger interface {
	Error(format string, v ...any)
	Warning(format string, v ...any)
	Info(format string, v ...any)
	Close() error
}

func New(path string, channel chan string) Logger {
	if path != "" {
		file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			log.Fatal(err)
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
		err:     log.New(os.Stderr, "", 0),
		wrn:     log.New(os.Stderr, "", 0),
		inf:     log.New(os.Stdout, "", 0),
		channel: channel,
	}
}

type StdLog struct {
	err, wrn, inf *log.Logger
	file          *os.File
	channel       chan string
}

func (l *StdLog) Error(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	_ = l.err.Output(2, fmt.Sprintf(format, v...))
	l.Send(msg)
}

func (l *StdLog) Info(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	_ = l.inf.Output(2, msg)
	l.Send(msg)
}

func (l *StdLog) Warning(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	_ = l.wrn.Output(2, msg)
	l.Send(msg)
}

func (l *StdLog) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *StdLog) Send(msg string) {
	if l.channel != nil {
		go func() {
			l.channel <- msg
		}()
	}
}

type EmptyLog struct{}

func NewEmptyLog() Logger { return EmptyLog{} }

func (l EmptyLog) Error(string, ...any)   {}
func (l EmptyLog) Warning(string, ...any) {}
func (l EmptyLog) Info(string, ...any)    {}
func (l EmptyLog) Close() error           { return nil }
