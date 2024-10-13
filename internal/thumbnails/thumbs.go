package thumbnails

import (
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/szonov/godlna/internal/backend"
	"github.com/szonov/godlna/internal/client"
	"github.com/szonov/godlna/internal/fs_util"
	"image"
	"image/color"
	"image/draw"
	"log/slog"
	"math/rand/v2"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

type ImageInfo struct {
	Name string
	Path string
	Mime string
	Time time.Time
	Size int64
}

func GetImageInfo(image string, profile *client.Profile) (*ImageInfo, error) {

	objectExt := path.Ext(image)
	objectID := image[:len(image)-len(objectExt)]
	objectPath := strings.Replace(objectID, "$", "/", -1)

	imagePath := path.Join(backend.CacheDir, "thumbs", objectPath, profile.Name+objectExt)

	var statInfo os.FileInfo
	var err error
	statInfo, err = os.Stat(imagePath)
	if err != nil && os.IsNotExist(err) {
		src := backend.GetObjectPath(objectID)
		if src == "" {
			return nil, fmt.Errorf("object path not found '%s'", objectID)
		}
		if err = makeThumb(src, imagePath, profile); err != nil {
			return nil, err
		}
		statInfo, err = os.Stat(imagePath)
		if err != nil {
			return nil, fmt.Errorf("generated thumb is not found")
		}
	}

	thumbMimeType := "image/jpeg"
	if strings.Contains(objectExt, "png") {
		thumbMimeType = "image/png"
	}
	return &ImageInfo{
		Name: statInfo.Name(),
		Path: imagePath,
		Mime: thumbMimeType,
		Time: statInfo.ModTime(),
		Size: statInfo.Size(),
	}, nil
}

func makeThumb(src, dest string, profile *client.Profile) (err error) {

	if err = fs_util.EnsureDirectoryExistsForFile(dest); err != nil {
		return err
	}

	// make thumbnail from video file, save it with extension .thumb
	args := make([]string, 0)
	args = append(args, "-s", "0", "-q", "10", "-c", "jpeg")
	args = append(args, "-t", "10")
	args = append(args, "-i", src, "-o", dest+".thumb")
	fmt.Printf("ICON ARGS: %v", args)
	cmd := exec.Command("ffmpegthumbnailer", args...)
	_, err = cmd.Output()
	if err != nil {
		slog.Error("makeVideoThumb",
			slog.String("cmd", "ffmpegthumbnailer "+strings.Join(cmd.Args, " ")),
			slog.String("err", err.Error()),
		)
	}

	// create now real final thumb with percents included
	percent := rand.IntN(101)

	err = MakeFinalThumb(dest+".thumb", dest, 480, profile.UseSquareThumbnails(), uint8(percent))

	if err != nil {
		slog.Error("makeFinalThumb",
			slog.String("dest", dest),
			slog.String("err", err.Error()),
		)
	}
	return
}

func MakeFinalThumb(src, dest string, thumbWidth int, squire bool, percent uint8) (err error) {

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
		percentWidth := int(percent) * thumbWidth / 100
		redRect := image.Rect(0, imageHeight-percentHeight, percentWidth, imageHeight)
		percentColor := color.RGBA{R: 110, G: 215, B: 92, A: 255}
		draw.Draw(dstImage, redRect, &image.Uniform{C: percentColor}, image.Point{}, draw.Src)
	}

	err = imaging.Save(dstImage, dest)
	return
}
