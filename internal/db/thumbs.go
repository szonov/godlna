package db

import (
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/szonov/godlna/internal/ffmpeg"
	"github.com/szonov/godlna/internal/fs_utils"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path"
)

func GetVideoThumb(objectID string) (thumbnailPath string, err error) {

	objCacheDir := getObjectCacheDir(objectID)
	videoFramePath := path.Join(objCacheDir, "video.jpg")
	thumbnailPath = path.Join(objCacheDir, "thumb.jpg")

	if _, err = os.Stat(thumbnailPath); err != nil && os.IsNotExist(err) {
		var object *Object

		if object = GetObject(objectID); object == nil {
			err = fmt.Errorf("object not found '%s'", objectID)
			return
		}

		if object.Type != TypeVideo {
			err = fmt.Errorf("object is not video '%s'", objectID)
			return
		}

		if object.Meta == nil {
			err = fmt.Errorf("object has no meta information '%s'", objectID)
			return
		}

		meta := object.Meta.(*VideoMeta)

		// by default 10% of full video duration (= duration / 10)
		thumbTimeSeek := meta.Duration.Divided(10)
		var watchedPercent uint8 = 0

		if object.Bookmark != nil {
			// TV set Bookmark = 0 when movie is watched
			// ... or when jump to bookmark and reset previous (but in this case next operation will be
			// one of: [1] set new bookmark or [2] set Bookmark = 0 - movie watched)
			if object.Bookmark.Uint64() == 0 {
				watchedPercent = 100
			} else {
				watchedPercent = object.Bookmark.PercentOf(meta.Duration)
				if watchedPercent == 0 {
					watchedPercent = 1
				}
				if watchedPercent > 0 && watchedPercent < 100 {
					thumbTimeSeek = object.Bookmark
				}
			}
		}

		if err = grabVideoFrame(object.FullPath(), videoFramePath, thumbTimeSeek.String()); err != nil {
			return
		}
		if err = makeThumbnail(videoFramePath, thumbnailPath, watchedPercent); err != nil {
			return
		}
	}
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

func makeThumbnail(src, dest string, watchedPercent uint8) (err error) {

	thumbWidth, thumbHeight := 480, 300
	coloredLineHeight := 20
	spaceAround := 0

	var srcImg image.Image
	var dstImage *image.NRGBA

	if srcImg, err = imaging.Open(src); err != nil {
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
