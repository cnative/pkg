package log

import (
	"testing"
)

func TestNewNop(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"succes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewNop()
			if got == nil {
				t.Errorf("NewNop() = nil, want  non-nil")
			}
		})
	}
}

func TestNew(t *testing.T) {
	type args struct {
		options []Option
	}
	tests := []struct {
		name    string
		args    args
		matcher func(l *logger) bool
	}{
		{
			"case-1", args{[]Option{WithFormat(JSON)}}, func(l *logger) bool {
				return l.format == JSON
			}},
		{
			"case-2", args{[]Option{WithTags(map[string]string{"environment": "dev", "version": "v1.2"})}}, func(l *logger) bool {
				return l.getEvironment() == "dev" && l.getVersion() == "v1.2"
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.args.options...)
			if tt.matcher != nil && !tt.matcher(got.(*logger)) {
				t.Error("New() = options mismatch")
			}
		})
	}
}

func TestLogger_NamedLogger(t *testing.T) {
	tests := []struct {
		name    string
		lName   string
		matcher func(l *logger) bool
	}{
		{
			"nammed-sub-logger", "sub-logger", func(l *logger) bool {
				return l.name == "sub-logger"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New()
			if got := l.NamedLogger(tt.lName).(*logger); !tt.matcher(got) {
				t.Errorf("Logger.NamedLogger() = %q, want %q", got.name, tt.lName)
			}
		})
	}
}
