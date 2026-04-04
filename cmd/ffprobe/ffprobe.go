package main

import (
	"fmt"
	"os"

	"github.com/szonov/godlna/pkg/ffprobe"
)

const (
	ExitSuccess         = 0
	ExitFileError       = 1
	ExitConfigureError  = 2
	ExitProcessingError = 3
	ExitDataError       = 4
)

func usage() {
	fmt.Printf("Usage: %s VIDEO_FILE\n", os.Args[0])
	os.Exit(ExitSuccess)
}

func main() {
	if len(os.Args) <= 1 {
		usage()
	}
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			usage()
		}
	}

	inputFile := os.Args[1]

	fmt.Printf("\nProcessing file: '%s'\n", inputFile)

	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		fmt.Printf("ERROR: file does not exist: %s\n", inputFile)
		os.Exit(ExitFileError)
	}

	if !ffprobe.Autodetect() {
		fmt.Printf("ERROR: ffprobe binary not found\n")
		os.Exit(ExitConfigureError)
	}

	data, err := ffprobe.Probe(inputFile)
	if err != nil {
		fmt.Printf("ERROR: ffprobe failed: %s %v\n", err, data)
		os.Exit(ExitProcessingError)
	}

	v := data.FirstVideoStream()
	if v == nil {
		fmt.Printf("ERROR: video stream not found\n")
		os.Exit(ExitDataError)
	}
	a := data.FirstAudioStream()
	if a == nil {
		fmt.Printf("ERROR: audio stream not found\n")
		os.Exit(ExitDataError)

	}
	fmt.Printf("\nFORMAT:\n")
	fmt.Printf("  name        : %s\n", data.Format.Name)
	fmt.Printf("  duration    : %s\n", data.Format.Duration)
	fmt.Printf("  size (bytes): %d\n", data.Format.Size)
	fmt.Printf("  bitrate     : %d\n", data.Format.BitRate)

	fmt.Printf("\nVIDEO:\n")
	fmt.Printf("  codec       : %s\n", v.CodecName)
	fmt.Printf("  channels    : %d\n", v.Channels)
	fmt.Printf("  sample rate : %d\n", v.SampleRate)
	fmt.Printf("  resolution  : %s\n", v.Resolution())

	fmt.Printf("\nAUDIO:\n")
	fmt.Printf("  codec       : %s\n", a.CodecName)
	fmt.Printf("  channels    : %d\n", a.Channels)
	fmt.Printf("  sample rate : %d\n", a.SampleRate)

	fmt.Printf("\n*DURATION ONLY:\n")
	dur, err := ffprobe.Duration(inputFile)
	if err != nil {
		fmt.Printf("  ERROR       : %s %v\n", err, data)
	} else {
		fmt.Printf("  duration    : %s\n", dur)
	}

	os.Exit(ExitSuccess)
}
