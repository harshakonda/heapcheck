package jsonproc

import (
	"testing"

	"github.com/harshakonda/heapcheck/guard"
)

func TestEncodeBad(t *testing.T) {
	defer guard.VerifyNone(t)

	event := SampleEvent()
	data, err := EncodeBad(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty data")
	}
}

func TestEncodeGood(t *testing.T) {
	defer guard.VerifyNone(t)

	event := SampleEvent()
	data, err := EncodeGood(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty data")
	}
}

func TestMarshalManual(t *testing.T) {
	defer guard.VerifyNone(t)

	event := SampleEvent()
	data := MarshalManual(event)
	if len(data) == 0 {
		t.Error("expected non-empty data")
	}
}

func TestNewEventBad(t *testing.T) {
	defer guard.VerifyNone(t)

	event := NewEventBad("info", "test message")
	if event.Level != "info" {
		t.Errorf("expected 'info', got '%s'", event.Level)
	}
}

func TestNewEventGood(t *testing.T) {
	defer guard.VerifyNone(t)

	event := NewEventGood("info", "test message")
	if event.Level != "info" {
		t.Errorf("expected 'info', got '%s'", event.Level)
	}

	event.AddField("key", "value")
	if event.Fields["key"] != "value" {
		t.Error("expected field to be set")
	}
}

func TestProcessStream(t *testing.T) {
	defer guard.VerifyNone(t)

	data := []byte(`[{"level":"error","message":"test"},{"level":"info","message":"ok"}]`)

	countBad, err := ProcessStreamBad(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if countBad != 1 {
		t.Errorf("expected 1, got %d", countBad)
	}

	countGood, err := ProcessStreamGood(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if countGood != 1 {
		t.Errorf("expected 1, got %d", countGood)
	}
}
