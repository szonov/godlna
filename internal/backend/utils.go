package backend

import (
	"path/filepath"
	"strconv"
	"strings"
)

var videoExts = []string{
	".mpg", ".mpeg", ".avi", ".mkv", ".mp4", ".m4v",
	".divx", ".asf", ".wmv", ".mts", ".m2ts", ".m2t",
	".vob", ".ts", ".flv", ".xvid", ".mov", ".3gp", ".rm", ".rmvb", ".webm",
}

func isVideoFile(file string) bool {
	fileExt := strings.ToLower(filepath.Ext(file))
	for _, ext := range videoExts {
		if fileExt == ext {
			return true
		}
	}
	return false
}

func NameWithoutExt(file string) string {
	ext := filepath.Ext(file)
	return file[0 : len(file)-len(ext)]
}

//func GetParentID(objectID string) string {
//	if pos := strings.LastIndex(objectID, "$"); pos != -1 {
//		return objectID[0:pos]
//	}
//	if objectID == "0" {
//		return "-1"
//	}
//	return "0"
//}

//	//time.Duration(f.DurationSeconds * float64(time.Second))

func FmtBitrate(bitRate string) uint {
	if v, err := strconv.ParseUint(bitRate, 10, 64); err == nil {
		if v > 8 {
			return uint(v / 8)
		}
		return uint(v)
	}
	return 0
}

//func fmtSampleRate(sampleRate string) int64 {
//	v, _ := strconv.ParseInt(sampleRate, 10, 64)
//	return v
//}
