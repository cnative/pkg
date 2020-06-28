package log

import (
	"testing"

	"go.uber.org/zap/zapcore"
)

func Test_newRollbarCore(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		environment string
		codeVersion string
		minLevel    Level
		matcher     func(zapcore.Core) bool
	}{
		{"matching log level", "some-token", "staging", "1.2", ErrorLevel, func(zc zapcore.Core) bool {
			return zc.Enabled(zapcore.ErrorLevel)
		}},
		{"not matching log level", "some-token", "staging", "1.2", ErrorLevel, func(zc zapcore.Core) bool {
			return !zc.Enabled(zapcore.InfoLevel)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newRollbarCore(tt.token, tt.environment, tt.codeVersion, tt.minLevel); !tt.matcher(got) {
				t.Errorf("newRollbarCore() = %v, mismatched", got)
			}
		})
	}
}

func Test_rollbarCore_Enabled(t *testing.T) {
	type args struct {
		l zapcore.Level
	}
	tests := []struct {
		name string
		r    *rollbarCore
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.Enabled(tt.args.l); got != tt.want {
				t.Errorf("rollbarCore.Enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
