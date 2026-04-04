package ffmpeg

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"os"
	"time"

	"github.com/szonov/godlna/pkg/imaging"
)

var DefaultTimeToSeekPercent = 10

type thumbnailConfig struct {
	width              int
	height             int
	completeLeeway     time.Duration
	jpegQuality        int
	progressBarOptions []imaging.ProgressBarOption
}

var defaultThumbnailConfig = thumbnailConfig{
	width:              480,
	height:             300,
	completeLeeway:     5 * time.Second,
	jpegQuality:        80,
	progressBarOptions: []imaging.ProgressBarOption{},
}

// ThumbnailOption sets an optional parameter for the making video thumbnail.
type ThumbnailOption func(*thumbnailConfig)

// Width returns an ThumbnailOption that sets the thumbnail width.
func Width(width int) ThumbnailOption {
	return func(c *thumbnailConfig) {
		c.width = width
	}
}

// Height returns an ThumbnailOption that sets the thumbnail height.
func Height(height int) ThumbnailOption {
	return func(c *thumbnailConfig) {
		c.height = height
	}
}

// CompleteLeeway returns an ThumbnailOption that sets the duration interval on the end of video file
// on which we think image is watched fully.
func CompleteLeeway(leeway time.Duration) ThumbnailOption {
	return func(c *thumbnailConfig) {
		c.completeLeeway = leeway
	}
}

// JPEGQuality returns an ThumbnailOption that sets the output JPEG quality.
// Quality ranges from 1 to 100 inclusive, higher is better. Default is 95.
func JPEGQuality(quality int) ThumbnailOption {
	return func(c *thumbnailConfig) {
		c.jpegQuality = quality
	}
}

// ProgressSize returns an ThumbnailOption that sets the height of progress bar if position top or bottom,
// or width of progress bar if position left or right.
func ProgressSize(size int) ThumbnailOption {
	return func(c *thumbnailConfig) {
		c.progressBarOptions = append(c.progressBarOptions, imaging.ProgressSize(size))
	}
}

// ProgressPaddingX returns an ThumbnailOption that sets the padding of the progress bar from left/right borders.
func ProgressPaddingX(padding int) ThumbnailOption {
	return func(c *thumbnailConfig) {
		c.progressBarOptions = append(c.progressBarOptions, imaging.ProgressPaddingX(padding))
	}
}

// ProgressPaddingY returns an ThumbnailOption that sets the padding of the progress bar from top/bottom borders.
func ProgressPaddingY(padding int) ThumbnailOption {
	return func(c *thumbnailConfig) {
		c.progressBarOptions = append(c.progressBarOptions, imaging.ProgressPaddingY(padding))
	}
}

// ProgressCompleteColor returns an ThumbnailOption that sets the color for completed part of progress bar.
func ProgressCompleteColor(cl color.Color) ThumbnailOption {
	return func(c *thumbnailConfig) {
		c.progressBarOptions = append(c.progressBarOptions, imaging.ProgressCompleteColor(cl))
	}
}

// ProgressIncompleteColor returns an ThumbnailOption that sets the color for incompleted part of progress bar.
func ProgressIncompleteColor(cl color.Color) ThumbnailOption {
	return func(c *thumbnailConfig) {
		c.progressBarOptions = append(c.progressBarOptions, imaging.ProgressIncompleteColor(cl))
	}
}

// ProgressFullColor returns an ThumbnailOption that sets the color for fully completed progress bar.
func ProgressFullColor(cl color.Color) ThumbnailOption {
	return func(c *thumbnailConfig) {
		c.progressBarOptions = append(c.progressBarOptions, imaging.ProgressFullColor(cl))
	}
}

// ProgressPosition returns an ThumbnailOption that sets position of the progress bar on image
func ProgressPosition(position imaging.Position) ThumbnailOption {
	return func(c *thumbnailConfig) {
		c.progressBarOptions = append(c.progressBarOptions, imaging.ProgressPosition(position))
	}
}

// ProgressPositionTop returns an ThumbnailOption that sets top position of the progress bar on image
func ProgressPositionTop() ThumbnailOption {
	return ProgressPosition(imaging.PositionTop)
}

// ProgressPositionRight returns an ThumbnailOption that sets right position of the progress bar on image
func ProgressPositionRight() ThumbnailOption {
	return ProgressPosition(imaging.PositionRight)
}

// ProgressPositionBottom returns an ThumbnailOption that sets bottom position of the progress bar on image
func ProgressPositionBottom() ThumbnailOption {
	return ProgressPosition(imaging.PositionBottom)
}

// ProgressPositionLeft returns an ThumbnailOption that sets left position of the progress bar on image
func ProgressPositionLeft() ThumbnailOption {
	return ProgressPosition(imaging.PositionLeft)
}

func Thumbnail(videoFile, thumbFile string, duration time.Duration, bookmark time.Duration, opts ...ThumbnailOption) error {
	var err error

	cfg := defaultThumbnailConfig
	for _, option := range opts {
		option(&cfg)
	}

	if _, err = os.Stat(videoFile); err != nil {
		return fmt.Errorf("video file not found '%s' (%w)", videoFile, err)
	}

	progress, timeToSeek := getProgressAndTimeToSeek(duration, bookmark, cfg.completeLeeway)

	var body []byte
	if body, err = GetVideoFrame(videoFile, timeToSeek); err != nil {
		return fmt.Errorf("can not get video frame from video '%s' (%w)", videoFile, err)
	}

	var im image.Image
	if im, _, err = image.Decode(bytes.NewReader(body)); err != nil {
		return fmt.Errorf("can not decode thumbnail to image object (%w)", err)
	}

	thumb := imaging.Thumbnail(im, cfg.width, cfg.height)
	imaging.AddProgressBar(thumb, progress, cfg.progressBarOptions...)

	return imaging.Save(thumb, thumbFile, cfg.jpegQuality)
}

func getProgressAndTimeToSeek(duration time.Duration, bookmark time.Duration, leeway time.Duration) (uint, time.Duration) {
	timeToSeek := duration / time.Duration(DefaultTimeToSeekPercent)
	var progress uint = 0

	if bookmark >= (duration - leeway) {
		progress = 100
	} else if bookmark > 0 && duration > 0 {
		timeToSeek = bookmark
		progress = uint(100 * bookmark / duration)
		if progress == 0 {
			progress = 1
		} else if progress > 100 {
			progress = 100
		}
	}
	return progress, timeToSeek
}
