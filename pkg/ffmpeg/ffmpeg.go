package ffmpeg

import (
	"fmt"
	"os/exec"
	"time"
)

var binPath = "ffmpeg"

// SetBinPath sets the global path to find and execute the `ffmpeg` program
func SetBinPath(path string) {
	binPath = path
}

// Autodetect try to find `ffmpeg` program in predefined paths
func Autodetect() bool {
	lookup := []string{
		"/var/packages/ffmpeg7/target/bin/ffmpeg",
		"/var/packages/ffmpeg6/target/bin/ffmpeg",
		"ffmpeg7",
		"ffmpeg6",
		"ffmpeg",
	}
	for _, p := range lookup {
		if val, err := exec.LookPath(p); err == nil {
			SetBinPath(val)
			return true
		}
	}
	return false
}

// DurationToString converts time.Duration value to string format accepted by `ffmpeg` program
func DurationToString(dur time.Duration) string {
	ms := dur.Milliseconds() % 1000
	s := int(dur.Seconds()) % 60
	m := int(dur.Minutes()) % 60
	h := int(dur.Hours())
	return fmt.Sprintf("%d:%02d:%02d.%03d", h, m, s, ms)
}

// GetVideoFrame captures a JPEG video frame from the timeToSeek position and returns it as binary content.
func GetVideoFrame(src string, timeToSeek time.Duration) ([]byte, error) {
	ss := DurationToString(timeToSeek)
	args := []string{"-ss", ss, "-i", src, "-y", "-r", "1", "-vframes", "1", "-an", "-loglevel", "panic", "-f", "mjpeg", "pipe:1"}
	cmd := exec.Command(binPath, args...)
	return cmd.Output()
}
