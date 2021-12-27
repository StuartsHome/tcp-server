package logger

import (
	"fmt"
	"log"
	"os"
	"path"
)

var logName = "numbers.log"

var ll Logger

type Logger struct {
	logger *log.Logger
}

func InitLogger(toFile bool) {
	ll.initLogger(toFile)
}

func (logger *Logger) initLogger(toFile bool) {
	if !toFile {
		ll = Logger{
			log.New(os.Stderr, "", 0),
		}
	} else {
		// If log file exists, remove it.
		if _, err := os.Stat(logName); err == nil {
			os.Remove(logName)
		}
		// Only create a new log file when toFile is true.
		file, err := logger.CreateLogFile(logName)
		if err != nil {
			return
		}
		ll = Logger{
			log.New(file, "", 0),
		}
	}
}

func (logger *Logger) CreateLogFile(file string) (*os.File, error) {
	folder, _ := path.Split(file)
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		os.Mkdir(folder, 0775)
	}

	return os.OpenFile(file,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}

func Log(message string) {
	ll.log(message)
}

func Logf(format string, messages ...interface{}) {
	ll.logf(format, messages...)
}

func Fatalf(format string, messages ...interface{}) {
	ll.logf(format, messages...)
	os.Exit(1)
}

func (logger Logger) log(message string) {
	logger.logger.Print(message)

}

func (logger Logger) logf(format string, messages ...interface{}) {
	logger.log(fmt.Sprintf(format, messages...))
}
