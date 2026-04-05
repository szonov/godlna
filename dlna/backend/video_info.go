package backend

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/szonov/godlna/pkg/ffprobe"
)

type VideoInfo struct {
	Format     string
	FileSize   int64
	VideoCodec string
	AudioCodec string
	Width      int
	Height     int
	Channels   int
	Bitrate    int
	Frequency  int
	Duration   int64
	Date       int64
}

func (mi *VideoInfo) readCacheFile(file string) error {
	body, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read: %w", err)
	}
	if err = json.Unmarshal(body, &mi); err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}
	return nil
}

func (mi *VideoInfo) parseVideoFile(file string) error {

	ffData, err := ffprobe.Probe(file)
	if err != nil {
		return fmt.Errorf("failed ffprobe '%s' : %w", file, err)
	}

	vStream := ffData.FirstVideoStream()
	aStream := ffData.FirstAudioStream()

	if vStream == nil || aStream == nil {
		return fmt.Errorf("video or audio stream is empty '%s'", file)
	}

	mi.Format = ffData.Format.Name
	//o.FileSize = f.Size()
	mi.VideoCodec = vStream.CodecName
	mi.AudioCodec = aStream.CodecName
	mi.Width = int(vStream.Width)
	mi.Height = int(vStream.Height)
	mi.Channels = int(vStream.Channels)
	mi.Bitrate = int(ffData.Format.BitRate)
	mi.Frequency = int(aStream.SampleRate)
	mi.Duration = ffData.Format.Duration.Milliseconds()
	//o.Date = f.ModTime().Unix()

	return nil
}

func (mi *VideoInfo) writeCacheFile(file string) error {
	// marshal to json
	body, err := json.Marshal(mi)
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}

	// create directory for file
	if err := os.MkdirAll(filepath.Dir(file), os.ModePerm); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create dir: %w", err)
	}

	// write file
	if err := os.WriteFile(file, body, 0666); err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}

	return nil
}

func videoInfoCacheFile(videoFile string) string {
	return filepath.Dir(videoFile) + "/@eaDir/" + filepath.Base(videoFile) + "/GODLNA_MEDIA_INFO"
}

func GetVideoInfo(videoFile string) (*VideoInfo, error) {

	cacheFile := videoInfoCacheFile(videoFile)
	mi := new(VideoInfo)

	info, err := os.Stat(videoFile)
	if err != nil {
		return nil, fmt.Errorf("(video_info) can not stat video file '%s': %w", videoFile, err)
	}

	videoModTime := info.ModTime().Unix()
	videoFileSize := info.Size()

	isValid := false
	if err := mi.readCacheFile(cacheFile); err != nil {
		//slog.Debug("(video_info) can not read cache file", "file", cacheFile, "err", err)
	} else if mi.FileSize == videoFileSize && mi.Date == videoModTime {
		isValid = true
	}

	if !isValid {
		if err := mi.parseVideoFile(videoFile); err != nil {
			return nil, fmt.Errorf("(video_info) can not parse video file '%s': %w", videoFile, err)
		}

		mi.FileSize = videoFileSize
		mi.Date = videoModTime

		if err := mi.writeCacheFile(cacheFile); err != nil {
			slog.Debug("(video_info) can not write cache file", "file", cacheFile, "err", err)
		}
	}

	return mi, nil
}
