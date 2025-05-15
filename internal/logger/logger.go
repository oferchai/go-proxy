package logger

import (
	"log"
	"os"
)

var logger *log.Logger

func Init(filename string) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	logger = log.New(file, "", log.LstdFlags)
	return nil
}

func Log(format string, v ...interface{}) {
	if logger != nil {
		logger.Printf(format, v...)
	}
}
