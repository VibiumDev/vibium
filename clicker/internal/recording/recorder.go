package recording

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// ScreenshotFunc is a function that captures a screenshot and returns base64-encoded PNG.
type ScreenshotFunc func() (string, error)

// Options configures the recorder.
type Options struct {
	// FPS is the frames per second for recording. Default: 10
	FPS int
	// OutputPath is where to save the video. If empty, uses temp directory.
	OutputPath string
	// Format is the output format: "mp4" or "webm". Default: "mp4"
	Format string
}

// Recorder captures screenshots at intervals and encodes them to video.
type Recorder struct {
	opts            Options
	screenshotFn    ScreenshotFunc
	tempDir         string
	frameCount      int
	mu              sync.Mutex
	stopChan        chan struct{}
	doneChan        chan struct{}
	running         bool
	outputPath      string
	captureErrors   []error
	lastCaptureBusy bool
}

// New creates a new Recorder.
func New(screenshotFn ScreenshotFunc, opts Options) *Recorder {
	if opts.FPS <= 0 {
		opts.FPS = 10
	}
	if opts.Format == "" {
		opts.Format = "mp4"
	}
	return &Recorder{
		opts:         opts,
		screenshotFn: screenshotFn,
	}
}

// Start begins recording.
func (r *Recorder) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return fmt.Errorf("recording already in progress")
	}

	// Create temp directory for frames
	tempDir, err := os.MkdirTemp("", "vibium-recording-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	r.tempDir = tempDir
	r.frameCount = 0
	r.captureErrors = nil
	r.stopChan = make(chan struct{})
	r.doneChan = make(chan struct{})
	r.running = true

	// Start capture goroutine
	go r.captureLoop()

	return nil
}

// Stop stops recording and encodes the video.
// Returns the path to the output video file.
func (r *Recorder) Stop() (string, error) {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return "", fmt.Errorf("no recording in progress")
	}
	r.running = false
	close(r.stopChan)
	r.mu.Unlock()

	// Wait for capture loop to finish
	<-r.doneChan

	// Get frame info under lock
	r.mu.Lock()
	frameCount := r.frameCount
	tempDir := r.tempDir
	captureErrors := make([]error, len(r.captureErrors))
	copy(captureErrors, r.captureErrors)
	r.mu.Unlock()

	if frameCount == 0 {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("no frames captured")
	}

	if len(captureErrors) > 0 {
		fmt.Printf("[recorder] %d capture errors occurred\n", len(captureErrors))
	}

	fmt.Printf("[recorder] Captured %d frames, encoding video...\n", frameCount)

	// Encode video
	outputPath, err := r.encode()
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to encode video: %w", err)
	}

	// Clean up temp directory
	os.RemoveAll(tempDir)

	r.outputPath = outputPath
	return outputPath, nil
}

// captureLoop captures screenshots at the configured FPS.
func (r *Recorder) captureLoop() {
	defer close(r.doneChan)

	ticker := time.NewTicker(time.Second / time.Duration(r.opts.FPS))
	defer ticker.Stop()

	for {
		select {
		case <-r.stopChan:
			return
		case <-ticker.C:
			r.captureFrame()
		}
	}
}

// captureFrame captures a single screenshot and saves it to the temp directory.
func (r *Recorder) captureFrame() {
	start := time.Now()
	base64Data, err := r.screenshotFn()
	elapsed := time.Since(start)
	fmt.Printf("[recorder] Screenshot took %v\n", elapsed)

	if err != nil {
		fmt.Printf("[recorder] Screenshot error: %v\n", err)
		r.mu.Lock()
		r.captureErrors = append(r.captureErrors, err)
		r.mu.Unlock()
		return
	}

	// Decode base64 to PNG bytes
	pngData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		fmt.Printf("[recorder] base64 decode error: %v\n", err)
		r.mu.Lock()
		r.captureErrors = append(r.captureErrors, err)
		r.mu.Unlock()
		return
	}

	// FIX: Capture vars under lock ONCE, then use locally
	var frameNum int
	var tempDir string
	r.mu.Lock()
	frameNum = r.frameCount
	tempDir = r.tempDir
	r.frameCount++
	r.mu.Unlock()

	// Now safe to use local copies
	framePath := filepath.Join(tempDir, fmt.Sprintf("frame_%06d.png", frameNum))
	fmt.Printf("[recorder] writing frame_%06d.png (%d bytes)\n", frameNum, len(pngData))

	if err := os.WriteFile(framePath, pngData, 0644); err != nil {
		fmt.Printf("[recorder] write error %s: %v\n", framePath, err)
		r.mu.Lock()
		r.captureErrors = append(r.captureErrors, err)
		r.mu.Unlock()
	}
}

// encode uses FFmpeg to encode the captured frames to video.
func (r *Recorder) encode() (string, error) {
	// Determine output path
	outputPath := r.opts.OutputPath
	if outputPath == "" {
		ext := r.opts.Format
		f, err := os.CreateTemp("", fmt.Sprintf("vibium-recording-*.%s", ext))
		if err != nil {
			return "", fmt.Errorf("failed to create output file: %w", err)
		}
		outputPath = f.Name()
		f.Close()
	}

	inputPattern := filepath.Join(r.tempDir, "frame_%06d.png")

	var args []string
	switch r.opts.Format {
	case "webm":
		args = []string{
			"-y",
			"-framerate", fmt.Sprintf("%d", r.opts.FPS),
			"-f", "image2",
			"-i", inputPattern,
			"-vf", "scale='trunc(iw/2)*2:trunc(ih/2)*2'", // Scale to even dimensions
			"-c:v", "libvpx-vp9",
			"-pix_fmt", "yuv420p",
			"-b:v", "2M",
			outputPath,
		}
	default: // mp4
		args = []string{
			"-y",
			"-framerate", fmt.Sprintf("%d", r.opts.FPS),
			"-f", "image2",
			"-i", inputPattern,
			"-vf", "scale='trunc(iw/2)*2:trunc(ih/2)*2'", // Scale to even dimensions
			"-c:v", "libx264",
			"-pix_fmt", "yuv420p",
			"-preset", "fast",
			"-crf", "23",
			outputPath,
		}
	}

	fmt.Printf("[recorder] running ffmpeg %v\n", args)
	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	fmt.Printf("[recorder] ffmpeg output:\n%s\n", string(output))

	if err != nil {
		return "", fmt.Errorf("ffmpeg failed: %w\nOutput: %s", err, string(output))
	}

	return outputPath, nil
}

// IsFFmpegAvailable checks if FFmpeg is installed and available.
func IsFFmpegAvailable() bool {
	cmd := exec.Command("ffmpeg", "-version")
	return cmd.Run() == nil
}
