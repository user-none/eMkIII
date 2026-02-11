//go:build !ios && !libretro

package ui

import (
	"testing"
	"time"
)

func TestSharedInput_SetAndRead(t *testing.T) {
	si := &SharedInput{}

	si.Set(true, false, true, false, true, false)
	si.SetPause()

	up, down, left, right, btn1, btn2, pause := si.Read()

	if !up || down || !left || right || !btn1 || btn2 {
		t.Fatalf("input mismatch: up=%v down=%v left=%v right=%v btn1=%v btn2=%v",
			up, down, left, right, btn1, btn2)
	}
	if !pause {
		t.Fatal("expected smsPause to be true")
	}

	// Pause should be cleared after read
	_, _, _, _, _, _, pause = si.Read()
	if pause {
		t.Fatal("expected smsPause to be cleared after read")
	}
}

func TestSharedFramebuffer_UpdateAndRead(t *testing.T) {
	sf := NewSharedFramebuffer()

	// Create some test pixel data
	stride := 256 * 4
	height := 192
	pixels := make([]byte, stride*height)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	sf.Update(pixels, stride, height, true)

	readPixels, readStride, readHeight, readLCB := sf.Read()

	if readStride != stride {
		t.Fatalf("stride mismatch: expected %d, got %d", stride, readStride)
	}
	if readHeight != height {
		t.Fatalf("height mismatch: expected %d, got %d", height, readHeight)
	}
	if !readLCB {
		t.Fatal("expected leftColumnBlank to be true")
	}

	// Verify pixel data (readPixels is a copy, safe to use)
	for i := 0; i < stride*height; i++ {
		if readPixels[i] != pixels[i] {
			t.Fatalf("pixel mismatch at %d: expected %d, got %d", i, pixels[i], readPixels[i])
		}
	}
}

func TestEmuControl_PauseResume(t *testing.T) {
	ec := NewEmuControl()

	// Start an emulation goroutine
	paused := make(chan struct{})
	resumed := make(chan struct{})
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			if !ec.CheckPause() {
				return
			}
			// Signal that we completed a CheckPause cycle
			select {
			case paused <- struct{}{}:
			default:
			}
			time.Sleep(time.Millisecond)
		}
	}()

	// Wait a bit for goroutine to start
	time.Sleep(20 * time.Millisecond)

	// Request pause (should block until ack)
	ec.RequestPause()

	if !ec.IsPaused() {
		t.Fatal("expected paused after RequestPause")
	}

	// Resume
	go func() {
		ec.RequestResume()
		close(resumed)
	}()
	<-resumed

	// Wait a bit for goroutine to resume
	time.Sleep(20 * time.Millisecond)

	if ec.IsPaused() {
		t.Fatal("expected not paused after RequestResume")
	}

	// Stop
	ec.Stop()
	<-done
}

func TestEmuControl_Stop(t *testing.T) {
	ec := NewEmuControl()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for ec.ShouldRun() {
			if !ec.CheckPause() {
				return
			}
			time.Sleep(time.Millisecond)
		}
	}()

	ec.Stop()

	select {
	case <-done:
		// Goroutine exited
	case <-time.After(time.Second):
		t.Fatal("goroutine did not exit after Stop")
	}
}

func TestEmuControl_StopWhilePaused(t *testing.T) {
	ec := NewEmuControl()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if !ec.CheckPause() {
				return
			}
			time.Sleep(time.Millisecond)
		}
	}()

	// Pause first
	ec.RequestPause()

	// Stop while paused — should unblock the goroutine
	ec.Stop()

	select {
	case <-done:
		// Goroutine exited
	case <-time.After(time.Second):
		t.Fatal("goroutine did not exit after Stop while paused")
	}
}

func TestEmuControl_DoubleRequestPause(t *testing.T) {
	ec := NewEmuControl()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if !ec.CheckPause() {
				return
			}
			time.Sleep(time.Millisecond)
		}
	}()

	// First pause
	ec.RequestPause()

	// Second pause should be a no-op (already paused)
	ec.RequestPause()

	if !ec.IsPaused() {
		t.Fatal("expected still paused")
	}

	ec.Stop()
	<-done
}
