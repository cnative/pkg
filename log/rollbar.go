package log

import (
	"encoding/json"
	"log"
	"os"

	"github.com/rollbar/rollbar-go"
	"go.uber.org/zap/zapcore"
)

// rollbarCore is a custom core to send logs to Rollbar
type rollbarCore struct {
	minLevel   zapcore.Level
	coreFields map[string]interface{}
}

// newRollbarCore sends logs to rollbar.
func newRollbarCore(token, environment, codeVersion string, minLevel Level) zapcore.Core {
	host, _ := os.Hostname()
	rollbar.SetToken(token)
	rollbar.SetEnvironment(environment)
	rollbar.SetServerHost(host)
	rollbar.SetCodeVersion(codeVersion)
	rollbar.SetEnabled(true)
	return &rollbarCore{
		minLevel:   zapcore.Level(minLevel),
		coreFields: make(map[string]interface{}),
	}
}

func (r *rollbarCore) Enabled(l zapcore.Level) bool {
	return l >= r.minLevel
}

// With provides structure
func (r *rollbarCore) With(fields []zapcore.Field) zapcore.Core {

	fieldMap := fieldsToMap(fields)

	for k, v := range fieldMap {
		r.coreFields[k] = v
	}

	return r
}

// Check if this should be sent to roll bar based
func (r *rollbarCore) Check(entry zapcore.Entry, checkedEntry *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if r.Enabled(entry.Level) {
		return checkedEntry.AddCore(entry, r)
	}

	return checkedEntry
}

func (r *rollbarCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {

	fieldMap := fieldsToMap(fields)

	if len(r.coreFields) > 0 {
		if coreFieldsMap, err := json.Marshal(r.coreFields); err != nil {
			log.Println("Unable to parse json for coreFields")
		} else {
			fieldMap["coreFields"] = string(coreFieldsMap)
		}
	}

	if entry.LoggerName != "" {
		fieldMap["logger"] = entry.LoggerName
	}
	if entry.Caller.TrimmedPath() != "" {
		fieldMap["file"] = entry.Caller.TrimmedPath()
	}
	if entry.Stack != "" {
		fieldMap["stack"] = entry.Stack
	}

	switch entry.Level {
	case zapcore.DebugLevel:
		rollbar.Debug(entry.Message, fieldMap)
	case zapcore.InfoLevel:
		rollbar.Info(entry.Message, fieldMap)
	case zapcore.WarnLevel:
		rollbar.Warning(entry.Message, fieldMap)
	case zapcore.ErrorLevel:
		rollbar.Critical(entry.Message, fieldMap)
	case zapcore.DPanicLevel:
		rollbar.Critical(entry.Message, fieldMap)
	case zapcore.PanicLevel:
		rollbar.Critical(entry.Message, fieldMap)
	case zapcore.FatalLevel:
		rollbar.Critical(entry.Message, fieldMap)
	}

	return nil
}

// Sync flushes
func (r *rollbarCore) Sync() error {
	rollbar.Wait()
	return nil
}

func fieldsToMap(fields []zapcore.Field) map[string]interface{} {
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range fields {
		f.AddTo(enc)
	}

	m := make(map[string]interface{})
	for k, v := range enc.Fields {
		m[k] = v
	}
	return m
}
