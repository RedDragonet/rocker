package pidlog

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

func formatWithPid(format string) string {
	pid := os.Getpid()

	return fmt.Sprintf("[%d] %s", pid, format)
}

func Info(args ...interface{}) {
	args = append([]interface{}{formatWithPid("")}, args...)
	log.Info(args...)
}

func Infof(format string, args ...interface{}) {
	log.Infof(formatWithPid(format), args...)
}

func Warnf(format string, args ...interface{}) {
	log.Infof(formatWithPid(format), args...)
}

func Errorf(format string, args ...interface{}) {
	log.Errorf(formatWithPid(format), args...)
}
