package patterns

import (
	"testing"

	"github.com/harshakonda/heapcheck/guard"
)

func TestNewUserBad(t *testing.T) {
	defer guard.VerifyNone(t)

	user := NewUserBad("test")
	if user.Name != "test" {
		t.Errorf("expected 'test', got '%s'", user.Name)
	}
}

func TestNewUserGood(t *testing.T) {
	defer guard.VerifyNone(t)

	user := NewUserGood("test")
	if user.Name != "test" {
		t.Errorf("expected 'test', got '%s'", user.Name)
	}
}

func TestNewUserWithStorage(t *testing.T) {
	defer guard.VerifyNone(t)

	var user User
	NewUserWithStorage(&user, "test")
	if user.Name != "test" {
		t.Errorf("expected 'test', got '%s'", user.Name)
	}
}

func TestLogFunctions(t *testing.T) {
	defer guard.VerifyNone(t)

	LogBad("test message")
	LogGood("test message")
	LogGeneric("test message")
	LogGeneric(42)
}

func TestClosureCapture(t *testing.T) {
	defer guard.VerifyNone(t,
		guard.MaxGoroutines(5),
		guard.SettleTime(100),
	)

	items := []string{"a", "b", "c"}
	ProcessGood(items)
}

func TestCollectFunctions(t *testing.T) {
	defer guard.VerifyNone(t)

	resultBad := CollectBad(100)
	if len(resultBad) != 100 {
		t.Errorf("expected 100, got %d", len(resultBad))
	}

	resultGood := CollectGood(100)
	if len(resultGood) != 100 {
		t.Errorf("expected 100, got %d", len(resultGood))
	}
}

func TestFormatID(t *testing.T) {
	defer guard.VerifyNone(t)

	bad := FormatIDBad(42)
	if bad != "42" {
		t.Errorf("expected '42', got '%s'", bad)
	}

	good := FormatIDGood(42)
	if good != "42" {
		t.Errorf("expected '42', got '%s'", good)
	}
}

func TestMapFunctions(t *testing.T) {
	defer guard.VerifyNone(t)

	m1 := CreateMapBad()
	if m1["key"] != 1 {
		t.Error("expected key=1")
	}

	m2 := CreateMapPooled()
	m2["test"] = 42
	ReturnMapToPool(m2)
}

func TestChannelSend(t *testing.T) {
	defer guard.VerifyNone(t)

	ch := make(chan User, 1)
	SendGood(ch)
	user := <-ch
	if user.Name != "test" {
		t.Errorf("expected 'test', got '%s'", user.Name)
	}
}

func TestStructSizes(t *testing.T) {
	defer guard.VerifyNone(t,
		guard.MaxHeapMB(1),
	)

	small := CreateSmall()
	_ = small
}
