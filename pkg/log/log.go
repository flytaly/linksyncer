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
}

func New() *StdLog {
	return &StdLog{
		err: log.New(os.Stderr, "", 0),
		wrn: log.New(os.Stderr, "", 0),
		inf: log.New(os.Stdout, "", 0),
	}
}

type StdLog struct {
	err, wrn, inf *log.Logger
}

func (l *StdLog) Error(format string, v ...any) {
	_ = l.err.Output(2, fmt.Sprintf(format, v...))
}

func (l *StdLog) Info(format string, v ...any) {
	_ = l.inf.Output(2, fmt.Sprintf(format, v...))
}

func (l *StdLog) Warning(format string, v ...any) {
	_ = l.wrn.Output(2, fmt.Sprintf(format, v...))
}

type EmptyLog struct{}

func NewEmptyLog() EmptyLog { return EmptyLog{} }

func (l EmptyLog) Error(string, ...any)   {}
func (l EmptyLog) Warning(string, ...any) {}
func (l EmptyLog) Info(string, ...any)    {}
