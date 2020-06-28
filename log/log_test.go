package log

import (
	"testing"
)

func TestNewNop(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"succes", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewNop()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNop() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
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
		matcher func(l *Logger) bool
	}{
		{
			"case-1", args{[]Option{WithFormat(JSON)}}, func(l *Logger) bool {
				return l.format == JSON
			}},
		{
			"case-2", args{[]Option{WithTags(map[string]string{"environment": "dev", "version": "v1.2"})}}, func(l *Logger) bool {
				return l.getEvironment() == "dev" && l.getVersion() == "v1.2"
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.options...)
			if err != nil {
				t.Errorf("New() error = %v, wantErr nil", err)
				return
			}
			if tt.matcher != nil && !tt.matcher(got) {
				t.Error("New() = options mismatch")
			}
		})
	}
}

func TestLogger_NamedLogger(t *testing.T) {
	tests := []struct {
		name    string
		lName   string
		matcher func(l *Logger) bool
	}{
		{
			"nammed-sub-logger", "sub-logger", func(l *Logger) bool {
				return l.name == "sub-logger"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := New()
			if err != nil {
				t.Errorf("New() error = %v, wantErr nil", err)
				return
			}
			if got := l.NamedLogger(tt.lName); !tt.matcher(got) {
				t.Errorf("Logger.NamedLogger() = %q, want %q", got.name, tt.lName)
			}
		})
	}
}
