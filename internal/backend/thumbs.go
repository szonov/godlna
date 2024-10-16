package backend

import (
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/szonov/godlna/internal/client"
	"github.com/szonov/godlna/internal/fs_util"
	"image"
	"image/color"
	"image/draw"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
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

		err = grabVideoFrame(object.FullPath(), videoFramePath, false)
		if err != nil {
			return "", time.Now(), err
		}

		err = makeThumbnail(videoFramePath, thumbnailPath, profile.UseSquareThumbnails(), object.BookmarkPercent())
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

func grabVideoFrame(src, dest string, force bool) (err error) {

	if force || !fs_util.FileExists(dest) {
		if err = fs_util.EnsureDirectoryExistsForFile(dest); err != nil {
			return err
		}

		cmd := exec.Command("ffmpegthumbnailer", "-s", "0", "-q", "10", "-c", "jpeg", "-t", "10",
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

func makeThumbnail(src, dest string, squire bool, bookmarkPercent uint8) (err error) {

	thumbWidth := 480
	coloredLineHeight := 20

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

	if bookmarkPercent > 0 {
		if bookmarkPercent > 100 {
			bookmarkPercent = 100
		}
		spaceAround := 10
		width := thumbWidth - 2*spaceAround
		coloredLineWidth := int(bookmarkPercent) * width / 100
		line := image.Rect(
			spaceAround, imageHeight-coloredLineHeight-spaceAround,
			spaceAround+coloredLineWidth, imageHeight-spaceAround,
		)
		barColor := color.RGBA{R: 110, G: 215, B: 92, A: 255}
		draw.Draw(dstImage, line, &image.Uniform{C: barColor}, image.Point{}, draw.Src)
	}

	err = imaging.Save(dstImage, dest)
	return
}
