package api

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
	"time"
)

func TestQueueActionScreenshotDoesNotBlockOnCapture(t *testing.T) {
	recorder := NewRecorder()
	recorder.Start(RecordingStartOptions{Screenshots: true}, nil)

	captureStarted := make(chan struct{})
	releaseCapture := make(chan struct{})
	recorder.StartActionScreenshotWorker(func(string, time.Time) {
		close(captureStarted)
		<-releaseCapture
	})

	start := time.Now()
	if !recorder.QueueActionScreenshot(time.Now(), 0, "page@1") {
		t.Fatal("QueueActionScreenshot returned false")
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("QueueActionScreenshot blocked for %s", elapsed)
	}

	select {
	case <-captureStarted:
	case <-time.After(time.Second):
		t.Fatal("queued screenshot was not processed")
	}

	close(releaseCapture)
	if !recorder.StopActionScreenshotWorker(time.Second) {
		t.Fatal("screenshot worker did not stop")
	}
}

func TestRecorderStopDrainsActionScreenshots(t *testing.T) {
	recorder := NewRecorder()
	recorder.Start(RecordingStartOptions{Screenshots: true, Format: "png"}, nil)
	recorder.StartActionScreenshotWorker(func(context string, ts time.Time) {
		recorder.AddScreenshot([]byte("fake-image"), context, 1, 1, ts)
	})

	if !recorder.QueueActionScreenshot(time.Now(), 0, "page@1") {
		t.Fatal("QueueActionScreenshot returned false")
	}

	zipData, err := recorder.Stop()
	if err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		t.Fatalf("recording zip parse failed: %v", err)
	}

	var sawFrame bool
	var sawResource bool
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "resources/") {
			sawResource = true
		}
		if strings.HasSuffix(f.Name, ".trace") {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("trace open failed: %v", err)
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("trace read failed: %v", err)
			}
			if strings.Contains(string(data), "screencast-frame") {
				sawFrame = true
			}
		}
	}

	if !sawFrame {
		t.Fatal("recording zip did not include queued screencast frame")
	}
	if !sawResource {
		t.Fatal("recording zip did not include queued screenshot resource")
	}
}
