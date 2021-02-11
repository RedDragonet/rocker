package pidlog

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

func init() {
	_, ok := os.LookupEnv("LOG_LEVEL")
	// LOG_LEVEL not set, let's default to debug
	if !ok {
		log.SetLevel(log.DebugLevel)
	}
}
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

func Debugf(format string, args ...interface{}) {
	log.Debugf(formatWithPid(format), args...)
}
