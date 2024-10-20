package store

import (
	"fmt"
	"github.com/szonov/godlna/internal/ffmpeg"
	"image"
	"image/color"
	"image/draw"
	"os"
	"time"

	"github.com/disintegration/imaging"
	"github.com/szonov/godlna/internal/fs_utils"
)

func GetThumbnail(objectID string, squire bool) (thumbnailPath string, t time.Time, err error) {

	t = time.Now()
	objCacheDir := getObjectCacheDir(objectID)
	videoFramePath := objCacheDir + "/video.jpg"
	if squire {
		thumbnailPath = objCacheDir + "/square.jpg"
	} else {
		thumbnailPath = objCacheDir + "/normal.jpg"
	}

	var statInfo os.FileInfo
	if statInfo, err = os.Stat(thumbnailPath); err != nil && os.IsNotExist(err) {
		var object *Object

		if object = GetObject(objectID); object == nil {
			err = fmt.Errorf("object not found '%s'", objectID)
			return
		}

		thumbTimeSeek := "10"
		var watchedPercent uint8 = 0

		if object.Bookmark != nil {
			// TV set Bookmark = 0 when movie is watched
			// ... or when jump to bookmark and reset previous (but in this case next operation will be
			// one of: [1] set new bookmark or [2] set Bookmark = 0 - movie watched)
			if object.Bookmark.Uint64() == 0 {
				watchedPercent = 100
			} else {
				watchedPercent = object.Bookmark.PercentOf(object.Duration)
				if watchedPercent > 0 && watchedPercent < 100 {
					thumbTimeSeek = object.Bookmark.String()
				}
			}
		}

		if err = grabVideoFrame(object.FullPath(), videoFramePath, thumbTimeSeek); err != nil {
			return
		}
		if err = makeThumbnail(videoFramePath, thumbnailPath, squire, watchedPercent); err != nil {
			return
		}
		if statInfo, err = os.Stat(thumbnailPath); err != nil {
			err = fmt.Errorf("generated thumb not found '%s' : %w", thumbnailPath, err)
			return
		}
	}
	t = statInfo.ModTime()
	return
}

func grabVideoFrame(src, dest string, timeToSeek string) (err error) {
	if !fs_utils.FileExists(dest) {
		if err = ffmpeg.MakeThumbnail(src, dest, timeToSeek); err != nil {
			err = fmt.Errorf("failed to generate thumbnail: %w (%s) (%s)", err, src, dest)
		}
	}
	return
}

func makeThumbnail(src, dest string, squire bool, watchedPercent uint8) (err error) {

	thumbWidth, thumbHeight := 480, 300
	coloredLineHeight := 30
	spaceAround := 0

	var srcImg image.Image
	var dstImage *image.NRGBA

	if srcImg, err = imaging.Open(src); err != nil {
		return err
	}
	//thumbWidth = 640
	if squire {
		thumbHeight = thumbWidth
		spaceAround = 80
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
