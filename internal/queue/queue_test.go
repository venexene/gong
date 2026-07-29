package queue

import (
	"errors"
	"testing"
	"time"

	"github.com/venexene/gong/internal/storage"
)

func TestCalcRetryDelay(t *testing.T) {
	tests := []struct {
		name     string
		retry    int
		expected time.Duration
	}{
		{"retry 1", 1, 5 * time.Second},
		{"retry 2", 2, 10 * time.Second},
		{"retry 3", 3, 20 * time.Second},
		{"retry 4", 4, 40 * time.Second},
		{"retry 5", 5, 80 * time.Second},
		{"retry 7", 7, 320 * time.Second},
		{"retry 8 hits cap", 8, MaxRetryDelay},
		{"retry 10 at cap", 10, MaxRetryDelay},
		{"retry 11 beyond max", 11, MaxRetryDelay},
		{"retry 100 far beyond max", 100, MaxRetryDelay},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcRetryDelay(tt.retry)
			if got != tt.expected {
				t.Errorf("calcRetryDelay(%d) = %v, want %v", tt.retry, got, tt.expected)
			}
		})
	}
}

func TestLogNotifierSend(t *testing.T) {
	n := storage.Notification{
		ID:      "test-id",
		Target:  "user@example.com",
		Message: "Hello, world!",
		Status:  "pending",
	}

	notifier := LogNotifier{}
	err := notifier.Send(n)
	if err != nil {
		t.Errorf("LogNotifier.Send() returned error: %v", err)
	}
}

func TestNotifierInterface(t *testing.T) {
	errNotifier := &errorNotifier{}

	n := storage.Notification{
		ID:     "err-id",
		Target: "fail@test.com",
	}

	err := errNotifier.Send(n)
	if err == nil {
		t.Error("expected error from errorNotifier, got nil")
	}
}

type errorNotifier struct{}

func (e *errorNotifier) Send(n storage.Notification) error {
	return errors.New("send failed")
}
