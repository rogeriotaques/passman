package selector

import (
	"bytes"
	"testing"
)

func TestVisibleRange_AllFit(t *testing.T) {
	start, end := visibleRange(0, 5, 15)
	if start != 0 || end != 5 {
		t.Errorf("expected [0,5), got [%d,%d)", start, end)
	}
}

func TestVisibleRange_Scrolled(t *testing.T) {
	start, end := visibleRange(10, 30, 10)
	if end-start != 10 {
		t.Errorf("expected window size 10, got %d", end-start)
	}
	if start < 0 || end > 30 {
		t.Errorf("out of bounds: [%d,%d)", start, end)
	}
	if 10 < start || 10 >= end {
		t.Errorf("cursor 10 not in visible range [%d,%d)", start, end)
	}
}

func TestVisibleRange_AtStart(t *testing.T) {
	start, end := visibleRange(0, 30, 10)
	if start != 0 {
		t.Errorf("expected start=0, got %d", start)
	}
	if end != 10 {
		t.Errorf("expected end=10, got %d", end)
	}
}

func TestVisibleRange_AtEnd(t *testing.T) {
	start, end := visibleRange(29, 30, 10)
	if end != 30 {
		t.Errorf("expected end=30, got %d", end)
	}
	if start != 20 {
		t.Errorf("expected start=20, got %d", start)
	}
}

func TestRender_HighlightsCursor(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{Items: []string{"alpha", "bravo", "charlie"}}
	render(&buf, opts, 1, 15)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("\x1b[7m bravo \x1b[0m")) {
		t.Errorf("expected bravo highlighted, got: %q", output)
	}
}

func TestRender_WithLabel(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{Label: "Pick one:", Items: []string{"a", "b"}}
	render(&buf, opts, 0, 15)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Pick one:")) {
		t.Errorf("expected label in output, got: %q", output)
	}
}

func TestMin(t *testing.T) {
	if min(3, 5) != 3 {
		t.Error("min(3,5) should be 3")
	}
	if min(5, 3) != 3 {
		t.Error("min(5,3) should be 3")
	}
}
