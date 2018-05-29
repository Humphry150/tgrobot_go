package bot

import (
	"bitbucket.org/magmeng/go-utils/log"
)

func Infof(format string, args ...interface{}) {
	log.InfoflnN(1, format, args...)
}

func Warnf(format string, args ...interface{}) {
	log.WarningflnN(1, format, args...)
}

func Errorf(format string, args ...interface{}) {
	log.ErrorflnN(1, format, args...)
}

func DebugLog(format string, args ...interface{}) {
	if *debugOn {
		log.InfoflnN(1, format, args...)
	}
}
