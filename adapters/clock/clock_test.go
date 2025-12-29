package clock_test

import (
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/clock"
)

func TestReal_Now(t *testing.T) {
	c := clock.Real{}

	before := time.Now()
	got := c.Now()
	after := time.Now()

	if got.Before(before) || got.After(after) {
		t.Errorf("Now() = %v, expected between %v and %v", got, before, after)
	}
}

func TestReal_Now_Successive(t *testing.T) {
	c := clock.Real{}

	t1 := c.Now()
	time.Sleep(time.Millisecond)
	t2 := c.Now()

	if !t2.After(t1) {
		t.Error("successive calls should return increasing time")
	}
}

func TestFake_NewFake(t *testing.T) {
	fixedTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	c := clock.NewFake(fixedTime)

	if c == nil {
		t.Fatal("expected non-nil clock")
	}

	got := c.Now()
	if !got.Equal(fixedTime) {
		t.Errorf("Now() = %v, want %v", got, fixedTime)
	}
}

func TestFake_Now_Stable(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	c := clock.NewFake(fixedTime)

	// Multiple calls should return same time
	for i := 0; i < 10; i++ {
		got := c.Now()
		if !got.Equal(fixedTime) {
			t.Errorf("call %d: Now() = %v, want %v", i, got, fixedTime)
		}
	}
}

func TestFake_Set(t *testing.T) {
	initial := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	c := clock.NewFake(initial)

	newTime := time.Date(2025, 12, 25, 10, 30, 0, 0, time.UTC)
	c.Set(newTime)

	got := c.Now()
	if !got.Equal(newTime) {
		t.Errorf("Now() = %v, want %v", got, newTime)
	}
}

func TestFake_Advance(t *testing.T) {
	initial := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	c := clock.NewFake(initial)

	c.Advance(time.Hour)
	got := c.Now()
	expected := initial.Add(time.Hour)

	if !got.Equal(expected) {
		t.Errorf("Now() = %v, want %v", got, expected)
	}
}

func TestFake_Advance_Multiple(t *testing.T) {
	initial := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	c := clock.NewFake(initial)

	c.Advance(time.Hour)
	c.Advance(30 * time.Minute)
	c.Advance(15 * time.Second)

	expected := initial.Add(time.Hour + 30*time.Minute + 15*time.Second)
	got := c.Now()

	if !got.Equal(expected) {
		t.Errorf("Now() = %v, want %v", got, expected)
	}
}

func TestFake_Advance_Negative(t *testing.T) {
	initial := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	c := clock.NewFake(initial)

	c.Advance(-time.Hour)
	got := c.Now()
	expected := initial.Add(-time.Hour)

	if !got.Equal(expected) {
		t.Errorf("Now() = %v, want %v", got, expected)
	}
}

func TestFake_ConcurrentAccess(t *testing.T) {
	c := clock.NewFake(time.Now())

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = c.Now()
				c.Advance(time.Second)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
	// Test passes if no race conditions
}
