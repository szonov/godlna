package indexer

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/disintegration/imaging"
	"image"
	"image/color"
	"image/draw"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

var ffmpegBinPath = "ffmpeg"

// SetFFMpegBinPath sets the global path to find and execute the `ffmpeg` program
func SetFFMpegBinPath(binPath string) {
	ffmpegBinPath = binPath
}

// FFMpegBinPathAutodetect try to find `ffmpeg` program in predefined paths
func FFMpegBinPathAutodetect() bool {
	lookup := []string{
		"/var/packages/ffmpeg7/target/bin/ffmpeg",
		"/var/packages/ffmpeg6/target/bin/ffmpeg",
		"ffmpeg7",
		"ffmpeg6",
		"ffmpeg",
	}
	for _, p := range lookup {
		if val, err := exec.LookPath(p); err == nil {
			SetFFMpegBinPath(val)
			return true
		}
	}
	return false
}

func GetVideoFrame(src, timeToSeek string) (body []byte, err error) {
	args := []string{"-ss", timeToSeek, "-i", src, "-y", "-r", "1", "-vframes", "1", "-an", "-loglevel", "panic", "-f", "mjpeg", "pipe:1"}
	cmd := exec.Command(ffmpegBinPath, args...)
	body, err = cmd.Output()
	return
}

// MakeThumbnailWithBookmark create thumbnail of video with orange/green line shown watch progress
func MakeThumbnailWithBookmark(videoFile, thumbFile string, duration int64, bookmark int64) (err error) {
	if _, err = os.Stat(videoFile); err != nil {
		err = fmt.Errorf("video file not found '%s' (%w)", videoFile, err)
		return
	}
	if err = EnsureDirectoryExistsForFile(thumbFile); err != nil {
		err = fmt.Errorf("can not create dir for thumbnail '%s' (%w)", thumbFile, err)
		return
	}

	thumbTimeSeek, watchedPercent := parseDurationAndBookmark(duration, bookmark)

	var body []byte
	if body, err = GetVideoFrame(videoFile, thumbTimeSeek); err != nil {
		return
	}
	err = transformAndSave(bytes.NewReader(body), thumbFile, watchedPercent)
	return
}

func parseDurationAndBookmark(duration int64, bookmark int64) (string, uint8) {
	if duration <= 0 {
		return DurationString(0), 0
	}
	if bookmark <= 0 {
		// by default 10% of full video duration
		return DurationString(duration / 10), 0
	}
	if bookmark >= duration {
		// by default 10% of full video duration
		return DurationString(duration / 10), 100
	}

	watchedPercent := uint8(100 * bookmark / duration)
	if watchedPercent == 0 {
		// make sure orange progress bar shown at least 1%
		watchedPercent = 1
	}
	return DurationString(bookmark), watchedPercent
}

func transformAndSave(r io.Reader, dest string, watchedPercent uint8) (err error) {
	thumbWidth, thumbHeight := 480, 300
	coloredLineHeight := 20
	spaceAround := 0

	var srcImg image.Image
	var dstImage *image.NRGBA

	if srcImg, _, err = image.Decode(r); err != nil {
		return err
	}
	dstImage = imaging.Thumbnail(srcImg, thumbWidth, thumbHeight, imaging.Lanczos)

	if watchedPercent > 0 {
		if watchedPercent > 100 {
			watchedPercent = 100
		}
		width := thumbWidth - 2*spaceAround
		coloredLineWidth := int(watchedPercent) * width / 100

		var line image.Rectangle
		var lineColor color.RGBA
		if watchedPercent < 100 {
			// draw gray background
			line = image.Rect(
				spaceAround, thumbHeight-coloredLineHeight,
				spaceAround+width, thumbHeight,
			)
			lineColor = color.RGBA{R: 106, G: 106, B: 106, A: 180}
			draw.Draw(dstImage, line, &image.Uniform{C: lineColor}, image.Point{X: 2, Y: 2}, draw.Over)

			lineColor = color.RGBA{R: 255, G: 85, B: 0, A: 255} // orange
		} else {
			lineColor = color.RGBA{R: 110, G: 215, B: 92, A: 255} // green
		}
		line = image.Rect(
			spaceAround, thumbHeight-coloredLineHeight,
			spaceAround+coloredLineWidth, thumbHeight,
		)
		draw.Draw(dstImage, line, &image.Uniform{C: lineColor}, image.Point{}, draw.Over)
	}

	err = imaging.Save(dstImage, dest)
	return
}

func MakeDSMStyleThumbnail(srcVideoFile string, duration int64, b sql.NullInt64, forceRecreate bool) {

	dir := filepath.Dir(srcVideoFile) + "/@eaDir/" + filepath.Base(srcVideoFile)
	im := dir + "/SYNOVIDEO_VIDEO_SCREENSHOT.jpg"

	if !FileExists(im) || forceRecreate {
		bookmark := b.Int64
		if bookmark == 0 && b.Valid {
			bookmark = duration
		}
		if bookmark > duration {
			bookmark = duration
		}
		if err := MakeThumbnailWithBookmark(srcVideoFile, im, duration, bookmark); err != nil {
			slog.Error(err.Error())
		}
	}
}
