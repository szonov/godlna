package imaging

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path"
	"strings"
)

// Save saves the image to a file (JPEG or PNG)
func Save(img image.Image, filename string, quality int) error {
	// create directory for output file
	dir, _ := path.Split(filename)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil && !os.IsExist(err) {
		return fmt.Errorf("can not create dir '%s' (%w)", dir, err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	if strings.HasSuffix(strings.ToLower(filename), ".png") {
		return png.Encode(file, img)
	}

	// Default to JPEG
	if quality <= 0 || quality > 100 {
		quality = 95
	}
	return jpeg.Encode(file, img, &jpeg.Options{Quality: quality})
}
