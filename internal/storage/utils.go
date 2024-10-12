package storage

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var videoExts = []string{
	".mpg", ".mpeg", ".avi", ".mkv", ".mp4", ".m4v",
	".divx", ".asf", ".wmv", ".mts", ".m2ts", ".m2t",
	".vob", ".ts", ".flv", ".xvid", ".mov", ".3gp", ".rm", ".rmvb", ".webm",
}

func isVideo(file string) bool {
	fileExt := strings.ToLower(filepath.Ext(file))

	for _, ext := range videoExts {
		if fileExt == ext {
			return true
		}
	}
	return false
}

func nameWithoutExt(file string) string {
	ext := filepath.Ext(file)
	return file[0 : len(file)-len(ext)]
}

func fmtDuration(d time.Duration) string {
	ms := d.Milliseconds() % 1000
	s := int(d.Seconds()) % 60
	m := int(d.Minutes()) % 60
	h := int(d.Hours())
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}

func fmtBitrate(bitRate string) int64 {
	if v, err := strconv.ParseInt(bitRate, 10, 64); err == nil {
		if v > 8 {
			return v / 8
		}
		return v
	}
	return 0
}
func fmtSampleRate(sampleRate string) int64 {
	v, _ := strconv.ParseInt(sampleRate, 10, 64)
	return v
}
