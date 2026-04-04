package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/szonov/godlna/pkg/ffmpeg"
	"github.com/szonov/godlna/pkg/ffprobe"
)

const (
	ExitSuccess         = 0
	ExitFileError       = 1
	ExitConfigureError  = 2
	ExitProcessingError = 3
)

func usage() {
	fmt.Printf("Usage: %s [OPTIONS] VIDEO_FILE\n\nOptions:\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(ExitSuccess)
}

func main() {
	var outputFile string
	var width int
	var height int
	var seekPercent int
	var progressSize int

	flag.StringVar(&outputFile, "output", "", "output `file` (default ${VIDEO_FILE}.jpg)")
	flag.IntVar(&width, "width", 480, "thumbnail width in `pixels`")
	flag.IntVar(&height, "height", 300, "thumbnail height in `pixels`")
	flag.IntVar(&seekPercent, "seek", 20, "watched `percent`, between 0 and 100")
	flag.IntVar(&progressSize, "progress-size", 10, "progress bar height in `pixels`, between 1 and ${HEIGHT}-1")

	if len(os.Args) <= 1 {
		usage()
	}
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			usage()
		}
	}
	flag.Parse()

	inputFile := flag.Arg(0)
	if inputFile == "" {
		usage()
	}

	videoFile, err := filepath.Abs(inputFile)
	if err != nil {
		fmt.Printf("ERROR: failed to get absolute path for input '%s': %w\n", videoFile, err)
		os.Exit(ExitFileError)
	}

	thumbFile := videoFile + ".jpg"
	if outputFile != "" {
		if thumbFile, err = filepath.Abs(outputFile); err != nil {
			fmt.Printf("ERROR: failed to get absolute path for output '%s': %w\n", outputFile, err)
			os.Exit(ExitFileError)
		}
	}

	fmt.Printf("Processing file: %s\n", videoFile)

	if _, err := os.Stat(videoFile); os.IsNotExist(err) {
		fmt.Printf("ERROR: file does not exist: %s\n", videoFile)
		os.Exit(ExitFileError)
	}

	if !ffmpeg.Autodetect() {
		fmt.Printf("ERROR: ffmpeg binary not found\n")
		os.Exit(ExitConfigureError)
	}

	if !ffprobe.Autodetect() {
		fmt.Printf("ERROR: ffprobe binary not found\n")
		os.Exit(ExitConfigureError)
	}

	duration, err := ffprobe.Duration(videoFile)
	if err != nil {
		fmt.Printf("ERROR: can not get duration: %s\n", err)
		os.Exit(ExitProcessingError)
	}

	var offset time.Duration = 0

	if seekPercent > 0 {
		if seekPercent > 100 {
			offset = duration
		} else {
			offset = duration * time.Duration(seekPercent) / 100
		}
	}

	if progressSize < 1 {
		progressSize = 1
	} else if progressSize > height-1 {
		progressSize = height - 1
	}

	fmt.Printf("\nINFO:\n")
	fmt.Printf("  input file : %s\n", videoFile)
	fmt.Printf("  thumbnail  : %s\n", thumbFile)
	fmt.Printf("  duration   : %s\n", duration)
	fmt.Printf("  offset     : %s\n", offset)

	options := []ffmpeg.ThumbnailOption{
		ffmpeg.Width(width),
		ffmpeg.Height(height),
		ffmpeg.CompleteLeeway(5 * time.Second),
		ffmpeg.JPEGQuality(80),
		ffmpeg.ProgressSize(progressSize),
		//ffmpeg.ProgressPaddingX(30),
		//ffmpeg.ProgressPaddingY(145),
		ffmpeg.ProgressPositionBottom(),
		//ffmpeg.ProgressCompleteColor(color.RGBA{R: 255, G: 0, B: 0, A: 255}),
	}

	if err = ffmpeg.Thumbnail(videoFile, thumbFile, duration, offset, options...); err != nil {
		fmt.Printf("ERROR: %s\n", err)
		os.Exit(ExitProcessingError)
	}

	os.Exit(ExitSuccess)
}
