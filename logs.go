package logs

import (
	"bytes"
	"camera-adapater/config"
	"fmt"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	log         = "LOG_CONFIG"
	withJson    = "WithJson"
	logLevel    = "LogLevel"
	logFile     = "LogFile"
	errLogFile  = "ErrLogFile"
	logRotation = "LogRotation"
)

func InitLog() {
	withjson, _ := config.Cfg.Bool(log, withJson)
	if withjson {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logrus.SetFormatter(new(LogFormatter))
	}

	loglevel, _ := config.Cfg.GetValue(log, logLevel)
	switch loglevel {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
		logrus.SetOutput(os.Stdout)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}

	log_file, _ := config.Cfg.GetValue(log, logFile)
	rotation, _ := config.Cfg.Bool(log, logRotation)
	if rotation {
		// 设置日志切割 大小200M 时间30天
		logrus.SetOutput(&lumberjack.Logger{
			Filename: log_file,
			MaxSize:  200,
			MaxAge:   30, //days
		})
	} else {
		os.MkdirAll(filepath.Dir(log_file), os.ModePerm)
		file, err := os.OpenFile(log_file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err == nil {
			logrus.SetOutput(file)
		}
	}
	logrus.AddHook(&ErrHook{})
	logrus.SetReportCaller(true)
}

// 自定义输出格式
type LogFormatter struct{}

func (*LogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	local, _ := time.LoadLocation("Asia/Shanghai")
	timestamp := time.Now().In(local).Format("2006-01-02 15:04:05")
	var file string
	var len int
	if entry.Caller != nil {
		file = filepath.Base(entry.Caller.File)
		len = entry.Caller.Line
	}
	msg := fmt.Sprintf("%s [%s:%d][GOID:%d][%s] %s\n", timestamp, file, len, getGID(), strings.ToUpper(entry.Level.String()), entry.Message)
	return []byte(msg), nil
}

// 获取协程id
func getGID() (n uint64) {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ = strconv.ParseUint(string(b), 10, 64)
	return
}

// 自定义hook，将异常日志单独写文件
type ErrHook struct {
}

func (*ErrHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.ErrorLevel,
		logrus.PanicLevel,
	}
}

func (*ErrHook) Fire(entry *logrus.Entry) error {
	log_file, _ := config.Cfg.GetValue(log, errLogFile)
	file, err := os.OpenFile(log_file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	if _, err := file.Write([]byte(entry.Message + "\n")); err != nil {
		return err
	}
	return nil
}
