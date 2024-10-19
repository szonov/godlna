package ffmpeg

import (
	"github.com/szonov/godlna/internal/fs_utils"
	"os/exec"
)

var ffmpegthumbnailerBinPath = "ffmpegthumbnailer"

// SetFFMpegThumbnailerBinPath sets the global path to find and execute the `ffmpegthumbnailer` program
func SetFFMpegThumbnailerBinPath(binPath string) {
	ffmpegthumbnailerBinPath = binPath
}

func MakeThumbnail(src, dest string, timeToSeek string) (err error) {
	if err = fs_utils.EnsureDirectoryExistsForFile(dest); err != nil {
		return err
	}
	args := []string{"-s", "0", "-q", "10", "-c", "jpeg", "-t", timeToSeek, "-i", src, "-o", dest}
	_, err = exec.Command(ffmpegthumbnailerBinPath, args...).Output()
	return
}
