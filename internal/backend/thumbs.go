package backend

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/szonov/godlna/internal/client"
	"github.com/szonov/godlna/internal/fs_util"
)

func objectThumbnailPaths(objectID string, profile *client.Profile) (thumbnailPath string, videoFramePath string) {
	objectPath := strings.Replace(objectID, "$", "/", -1)
	thumbnailPath = path.Join(CacheDir, "thumbs", objectPath, profile.Name+".jpg")
	videoFramePath = path.Join(CacheDir, "thumbs", objectPath, "video-frame.jpg")
	return
}

func GetThumbnail(objectID string, profile *client.Profile) (imPath string, t time.Time, err error) {

	thumbnailPath, videoFramePath := objectThumbnailPaths(objectID, profile)

	var statInfo os.FileInfo
	if statInfo, err = os.Stat(thumbnailPath); err != nil && os.IsNotExist(err) {
		var object *Object

		if object = GetObject(objectID); object == nil {
			return "", time.Now(), fmt.Errorf("object not found '%s'", objectID)
		}

		watchedPercent := object.Bookmark.PercentOf(object.Duration)
		thumbTimeSeek := "10"
		if watchedPercent > 0 && watchedPercent < 100 {
			thumbTimeSeek = object.Bookmark.String()
		}

		err = grabVideoFrame(object.FullPath(), videoFramePath, thumbTimeSeek)
		if err != nil {
			return "", time.Now(), err
		}

		err = makeThumbnail(videoFramePath, thumbnailPath, profile.UseSquareThumbnails(), watchedPercent)
		if err != nil {
			return "", time.Now(), err
		}
		statInfo, err = os.Stat(thumbnailPath)
		if err != nil {
			return "", time.Now(), fmt.Errorf("generated thumb not found '%s'", thumbnailPath)
		}
	}

	return thumbnailPath, statInfo.ModTime(), nil
}

func grabVideoFrame(src, dest string, timeSeek string) (err error) {

	if !fs_util.FileExists(dest) {
		slog.Debug("grabVideoFrame", "src", src, "dest", dest, "timeSeek", timeSeek)

		if err = fs_util.EnsureDirectoryExistsForFile(dest); err != nil {
			return err
		}

		cmd := exec.Command("ffmpegthumbnailer", "-s", "0", "-q", "10", "-c", "jpeg", "-t", timeSeek,
			"-i", src, "-o", dest)

		if _, err = cmd.Output(); err != nil {
			slog.Error("grabVideoFrame",
				slog.String("cmd", "ffmpegthumbnailer "+strings.Join(cmd.Args, " ")),
				slog.String("err", err.Error()),
			)
		}
	}
	return
}

func makeThumbnail(src, dest string, squire bool, watchedPercent uint8) (err error) {

	thumbWidth := 480
	coloredLineHeight := 20
	spaceAround := 0

	var srcImg image.Image
	var dstImage *image.NRGBA

	if srcImg, err = imaging.Open(src); err != nil {
		return err
	}

	if squire {
		dstImage = imaging.Thumbnail(srcImg, thumbWidth, thumbWidth, imaging.Lanczos)
	} else {
		dstImage = imaging.Resize(srcImg, thumbWidth, 0, imaging.Lanczos)
	}

	imageHeight := dstImage.Bounds().Max.Y

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
				spaceAround, imageHeight-coloredLineHeight-spaceAround,
				spaceAround+width, imageHeight-spaceAround,
			)
			lineColor = color.RGBA{R: 106, G: 106, B: 106, A: 180}
			draw.Draw(dstImage, line, &image.Uniform{C: lineColor}, image.Point{2, 2}, draw.Over)

			lineColor = color.RGBA{R: 255, G: 85, B: 0, A: 255} // orange
		} else {
			lineColor = color.RGBA{R: 110, G: 215, B: 92, A: 255} // green
		}
		line = image.Rect(
			spaceAround, imageHeight-coloredLineHeight-spaceAround,
			spaceAround+coloredLineWidth, imageHeight-spaceAround,
		)
		draw.Draw(dstImage, line, &image.Uniform{C: lineColor}, image.Point{}, draw.Over)
	}

	err = imaging.Save(dstImage, dest)
	return
}

func removeThumbnails(objectID string) {
	dir := path.Join(CacheDir, "thumbs", strings.Replace(objectID, "$", "/", -1))
	slog.Debug("REMOVE", "dir", dir)
	err := os.RemoveAll(dir)
	if err != nil {
		return
	}
}
