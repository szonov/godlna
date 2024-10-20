package ffmpeg

import (
	"github.com/szonov/godlna/internal/fs_utils"
	"os/exec"
)

var ffmpegBinPath = "ffmpeg"

// SetFFMpegBinPath sets the global path to find and execute the `ffmpeg` program
func SetFFMpegBinPath(binPath string) {
	ffmpegBinPath = binPath
}

func MakeThumbnail(src, dest string, timeToSeek string) (err error) {
	if err = fs_utils.EnsureDirectoryExistsForFile(dest); err != nil {
		return err
	}
	args := []string{"-ss", timeToSeek, "-i", src, "-y", "-r", "1", "-vframes", "1", "-an", "-loglevel", "panic", dest}
	cmd := exec.Command(ffmpegBinPath, args...)
	_, err = cmd.Output()
	return
}
