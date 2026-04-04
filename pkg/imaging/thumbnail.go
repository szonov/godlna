package imaging

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

// Thumbnail creates a thumbnail of the specified size by scaling and cropping the image
// to the target dimensions while preserving the central part.
func Thumbnail(src image.Image, width, height int) *image.RGBA {
	srcBounds := src.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	// Calculate aspect ratios
	srcAspect := float64(srcWidth) / float64(srcHeight)
	dstAspect := float64(width) / float64(height)

	var (
		scaleWidth, scaleHeight int
		cropX, cropY            int
	)

	// Determine how to scale so that one side exactly matches the target size,
	// and the other side is larger or equal
	if srcAspect > dstAspect {
		// Source image is wider - fit by height
		scaleHeight = height
		scaleWidth = int(float64(height) * srcAspect)
		cropX = (scaleWidth - width) / 2
		cropY = 0
	} else {
		// Source image is taller - fit by width
		scaleWidth = width
		scaleHeight = int(float64(width) / srcAspect)
		cropX = 0
		cropY = (scaleHeight - height) / 2
	}

	// Create scaled image
	scaled := scaleImage(src, scaleWidth, scaleHeight)

	// Create final image and crop the central part
	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	// Draw the required area from the scaled image
	draw.Draw(dst, dst.Bounds(), scaled, image.Point{cropX, cropY}, draw.Src)

	return dst
}

// scaleImage scales the image to the specified dimensions using bilinear interpolation
func scaleImage(src image.Image, newWidth, newHeight int) *image.RGBA {
	srcBounds := src.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Scaling factors
	scaleX := float64(srcWidth) / float64(newWidth)
	scaleY := float64(srcHeight) / float64(newHeight)

	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			// Find corresponding position in the source image
			srcX := float64(x) * scaleX
			srcY := float64(y) * scaleY

			// Bilinear interpolation
			dst.Set(x, y, bilinearInterpolate(src, srcX, srcY))
		}
	}

	return dst
}

// bilinearInterpolate performs bilinear interpolation for a point at coordinates (x, y)
func bilinearInterpolate(src image.Image, x, y float64) color.Color {
	x1, y1 := int(math.Floor(x)), int(math.Floor(y))
	x2, y2 := x1+1, y1+1

	// Handle edge cases
	if x2 >= src.Bounds().Dx() {
		x2 = x1
	}
	if y2 >= src.Bounds().Dy() {
		y2 = y1
	}

	// Weights for interpolation
	wx := x - float64(x1)
	wy := y - float64(y1)

	// Get colors of four neighboring pixels
	c11 := color.RGBAModel.Convert(src.At(x1, y1)).(color.RGBA)
	c12 := color.RGBAModel.Convert(src.At(x1, y2)).(color.RGBA)
	c21 := color.RGBAModel.Convert(src.At(x2, y1)).(color.RGBA)
	c22 := color.RGBAModel.Convert(src.At(x2, y2)).(color.RGBA)

	// X interpolation for top and bottom rows
	r1 := float64(c11.R)*(1-wx) + float64(c21.R)*wx
	g1 := float64(c11.G)*(1-wx) + float64(c21.G)*wx
	b1 := float64(c11.B)*(1-wx) + float64(c21.B)*wx
	a1 := float64(c11.A)*(1-wx) + float64(c21.A)*wx

	r2 := float64(c12.R)*(1-wx) + float64(c22.R)*wx
	g2 := float64(c12.G)*(1-wx) + float64(c22.G)*wx
	b2 := float64(c12.B)*(1-wx) + float64(c22.B)*wx
	a2 := float64(c12.A)*(1-wx) + float64(c22.A)*wx

	// Y interpolation
	r := r1*(1-wy) + r2*wy
	g := g1*(1-wy) + g2*wy
	b := b1*(1-wy) + b2*wy
	a := a1*(1-wy) + a2*wy

	return color.RGBA{
		R: uint8(math.Round(r)),
		G: uint8(math.Round(g)),
		B: uint8(math.Round(b)),
		A: uint8(math.Round(a)),
	}
}
