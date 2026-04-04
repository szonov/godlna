package imaging

import (
	"image"
	"image/color"
	"image/draw"
)

type Position int

const (
	PositionTop Position = iota
	PositionRight
	PositionBottom
	PositionLeft
)

type progressBarConfig struct {
	size            int
	paddingX        int
	paddingY        int
	position        Position
	completeColor   color.Color
	incompleteColor color.Color
	fullColor       color.Color
}

var defaultProgressBarConfig = progressBarConfig{
	size:            15,
	paddingX:        0,
	paddingY:        0,
	position:        PositionBottom,
	completeColor:   color.RGBA{R: 255, G: 85, B: 0, A: 255},    // orange
	incompleteColor: color.RGBA{R: 106, G: 106, B: 106, A: 180}, // gray with opacity
	fullColor:       color.RGBA{R: 110, G: 215, B: 92, A: 255},  // green
}

// ProgressBarOption sets an optional parameter for the adding progress bar to image.
type ProgressBarOption func(*progressBarConfig)

// ProgressSize returns an ProgressBarOption that sets the height of progress bar if position top or bottom,
// or width of progress bar if position left or right.
func ProgressSize(size int) ProgressBarOption {
	return func(c *progressBarConfig) {
		c.size = size
	}
}

// ProgressPaddingX returns an ProgressBarOption that sets the padding of the progress bar from left/right borders.
func ProgressPaddingX(padding int) ProgressBarOption {
	return func(c *progressBarConfig) {
		c.paddingX = padding
	}
}

// ProgressPaddingY returns an ProgressBarOption that sets the padding of the progress bar from top/bottom borders.
func ProgressPaddingY(padding int) ProgressBarOption {
	return func(c *progressBarConfig) {
		c.paddingY = padding
	}
}

// ProgressCompleteColor returns an ProgressBarOption that sets the color for completed part of progress bar
func ProgressCompleteColor(cl color.Color) ProgressBarOption {
	return func(c *progressBarConfig) {
		c.completeColor = cl
	}
}

// ProgressIncompleteColor returns an ProgressBarOption that sets the color incompleted part of progress bar.
func ProgressIncompleteColor(cl color.Color) ProgressBarOption {
	return func(c *progressBarConfig) {
		c.incompleteColor = cl
	}
}

// ProgressFullColor returns an ProgressBarOption that sets the color for fully (100%) completed progress.
func ProgressFullColor(cl color.Color) ProgressBarOption {
	return func(c *progressBarConfig) {
		c.fullColor = cl
	}
}

// ProgressPosition returns an ProgressBarOption that sets position of the progress bar on image
func ProgressPosition(position Position) ProgressBarOption {
	return func(c *progressBarConfig) {
		if position >= PositionTop && position <= PositionLeft {
			c.position = position
		}
	}
}

func AddProgressBar(im *image.RGBA, progress uint, opts ...ProgressBarOption) {
	if im == nil {
		return
	}
	cfg := defaultProgressBarConfig
	for _, option := range opts {
		option(&cfg)
	}

	bounds := im.Bounds()
	imageWidth, imageHeight := bounds.Dx(), bounds.Dy()

	if progress > 0 {
		if progress > 100 {
			progress = 100
		}

		var rect image.Rectangle
		var rectColor color.Color
		if progress < 100 {
			// draw gray background
			rect = getRectangle(100, imageWidth, imageHeight, cfg)
			draw.Draw(im, rect, &image.Uniform{C: cfg.incompleteColor}, image.Point{X: 2, Y: 2}, draw.Over)
			rectColor = cfg.completeColor // orange
		} else {
			rectColor = cfg.fullColor // green
		}
		rect = getRectangle(int(progress), imageWidth, imageHeight, cfg)
		draw.Draw(im, rect, &image.Uniform{C: rectColor}, image.Point{}, draw.Over)
	}
}

func getRectangle(progress, imageWidth, imageHeight int, cfg progressBarConfig) image.Rectangle {
	pos := cfg.position
	var size int

	if pos == PositionTop || pos == PositionBottom {
		size = (imageWidth - 2*cfg.paddingX) * progress / 100
	} else {
		size = (imageHeight - 2*cfg.paddingY) * progress / 100
	}

	switch pos {
	case PositionLeft:
		return image.Rect(
			cfg.paddingX, imageHeight-cfg.paddingY,
			cfg.paddingX+cfg.size, imageHeight-cfg.paddingY-size,
		)
	case PositionRight:
		return image.Rect(
			imageWidth-cfg.paddingX-cfg.size, imageHeight-cfg.paddingY,
			imageWidth-cfg.paddingX, imageHeight-cfg.paddingY-size,
		)
	case PositionTop:
		return image.Rect(
			cfg.paddingX, cfg.paddingY,
			cfg.paddingX+size, cfg.paddingY+cfg.size,
		)
	case PositionBottom:
		return image.Rect(
			cfg.paddingX, imageHeight-cfg.paddingY,
			cfg.paddingX+size, imageHeight-cfg.paddingY-cfg.size,
		)
	default:
		panic("unhandled default case")
	}
}
