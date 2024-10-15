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

type ThumbnailInfo struct {
	Path string
	Time time.Time
}

func GetThumbnailInfo(objectID string, profile *client.Profile) (*ThumbnailInfo, error) {
	objectPath := strings.Replace(objectID, "$", "/", -1)
	imagePath := path.Join(CacheDir, "thumbs", objectPath, profile.Name+".jpg")

	var statInfo os.FileInfo
	var err error
	statInfo, err = os.Stat(imagePath)
	if err != nil && os.IsNotExist(err) {
		src, percent, seen := GetObjectPathPercentSeen(objectID)
		slog.Debug("PPP",
			"src", src,
			"percent", percent,
			"seen", seen,
		)
		if src == "" {
			return nil, fmt.Errorf("object path not found '%s'", objectID)
		}
		if err = makeThumb(src, imagePath, percent, seen, profile); err != nil {
			return nil, err
		}
		statInfo, err = os.Stat(imagePath)
		if err != nil {
			return nil, fmt.Errorf("generated thumb is not found")
		}
	}

	return &ThumbnailInfo{Path: imagePath, Time: statInfo.ModTime()}, nil
}

func makeThumb(src, dest string, percent uint8, seen bool, profile *client.Profile) (err error) {

	videoThumb := NameWithoutExt(dest) + "-orig.jpg"

	if !fs_util.FileExists(videoThumb) {
		if err = fs_util.EnsureDirectoryExistsForFile(dest); err != nil {
			return err
		}
		cmd := exec.Command("ffmpegthumbnailer",
			"-s", "0", "-q", "10", "-c", "jpeg", "-t", "10",
			"-i", src, "-o", videoThumb)
		if _, err = cmd.Output(); err != nil {
			slog.Error("makeVideoThumb",
				slog.String("cmd", "ffmpegthumbnailer "+strings.Join(cmd.Args, " ")),
				slog.String("err", err.Error()),
			)
			return
		}
	}

	// create now real final thumb with percents included
	err = makeFinalThumb(videoThumb, dest, 480, profile.UseSquareThumbnails(), percent)

	if err != nil {
		slog.Error("makeFinalThumb", slog.String("dest", dest), slog.String("err", err.Error()))
	}
	return
}

func makeFinalThumb(src, dest string, thumbWidth int, squire bool, percent uint8) (err error) {

	percentHeight := 40
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

	if percent > 0 {
		if percent > 100 {
			percent = 100
		}
		spaceAround := 10
		barWidth := thumbWidth - 2*spaceAround
		percentWidth := int(percent) * barWidth / 100
		bar := image.Rect(spaceAround, imageHeight-percentHeight, percentWidth, imageHeight-spaceAround)
		barColor := color.RGBA{R: 110, G: 215, B: 92, A: 255}
		draw.Draw(dstImage, bar, &image.Uniform{C: barColor}, image.Point{}, draw.Src)
	}

	err = imaging.Save(dstImage, dest)
	return
}
