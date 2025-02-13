package log

import (
	"os"
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestLog_ErrorOutput(t *testing.T) {
	errorlog := newZaplogger(os.Stdout, InfoLevel, JsonEncoder,
		AddCaller(),
		ErrorOutput(zapcore.AddSync(OpenLogFile("error.log"))),
	)

	defer errorlog.Sync()
	errorlog.Error("error test log")
	errorlog.Error("fatal")
	errorlog.Info("info test log")
	errorlog.Warn("warn")
	errorlog.Debug("debug")
}
