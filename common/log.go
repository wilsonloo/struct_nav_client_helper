// elogger
package common

import (
	"fmt"

	"log"
)

const (
	LOG_LEVEL_NOT_SET int = iota
	LOG_LEVEL_DEBUG
	LOG_LEVEL_INFO
	LOG_LEVEL_WARNING
	LOG_LEVEL_ERROR

	LOG_LEVEL_MAX
)

// log 等级
var g_log_level int

func SetLogLevel(lvl int) {
	if lvl >= LOG_LEVEL_NOT_SET && lvl < LOG_LEVEL_MAX {
		g_log_level = lvl
	} else {
		fmt.Println("invalid log level: ", lvl)
	}
}

func LogDebug(args ...interface{}) {
	if g_log_level > LOG_LEVEL_DEBUG {
		return
	}

	log.Println(args)
}

func LogInfo(args ...interface{}) {
	if g_log_level > LOG_LEVEL_INFO {
		return
	}

	log.Println(args)
}

func LogWarning(args ...interface{}) {
	if g_log_level > LOG_LEVEL_WARNING {
		return
	}

	log.Println(args)
}

func LogError(args ...interface{}) {
	if g_log_level > LOG_LEVEL_ERROR {
		return
	}

	log.Println(args)
}
