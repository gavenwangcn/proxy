package util

import (
	"os"
	"strings"

	"github.com/CodisLabs/codis/pkg/utils/bytesize"
	"github.com/CodisLabs/codis/pkg/utils/log"
	l4g "github.com/alecthomas/log4go"
)

var (
	maxFileFrag       = 1000000
	maxFragSize int64 = bytesize.MB * 100
)

// SetLogLevel set log level
func SetLogLevel(level string) string {
	level = strings.ToLower(level)
	var l = log.LEVEL_INFO
	switch level {
	case "error":
		l = log.LEVEL_ERROR
	case "warn", "warning":
		l = log.LEVEL_WARN
	case "debug":
		l = log.LEVEL_DEBUG
	case "info":
		fallthrough
	default:
		level = "info"
		l = log.LEVEL_INFO
	}
	log.SetLevel(l)
	log.Infof("set log level to <%s>", level)

	return level
}

// InitLog init log
func InitLog(file string) {
	// set output log file
	if "" != file {
		f, err := log.NewRollingFile(file, maxFileFrag, maxFragSize)
		if err != nil {
			log.PanicErrorf(err, "open rolling log file failed: %s", file)
		} else {
			log.StdLog = log.New(f, "")
		}
	}

	log.SetLevel(log.LEVEL_INFO)
	log.SetFlags(log.Flags() | log.Lshortfile)
}

func InitAccessLog(file string) {
	//set output access log file
	if "" != file {
		fileLog := l4g.NewFileLogWriter(file, true)
		//access日志只打印出自定义message信息
		fileLog.SetFormat("%M").SetRotateSize(int(maxFragSize)).SetRotateLines(maxFileFrag).SetRotateHour(true)

		l4g.AddFilter("file", l4g.INFO, fileLog)
		l4g.AddFilter("stdout", l4g.ERROR, l4g.NewConsoleLogWriter())
	} else {
		std := os.Stdout
		l4g.AddFilter("stdout", l4g.INFO, l4g.NewFormatLogWriter(std, "%M"))
	}

}
