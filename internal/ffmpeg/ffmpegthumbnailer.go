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
	//args := []string{"-i", src, "-y", "-an", "-ss", timeToSeek, "-an", "-r", "1", "-vframes", "1", "-loglevel", "panic", dest}

	//ffmpeg6 -i ... -y -an -ss 00:01:10 -an -r 1 -vframes 1 -loglevel panic one2.jpg
	_, err = exec.Command(ffmpegthumbnailerBinPath, args...).Output()
	//args := []string{"-i", src, "-y", "-an", "-ss", timeToSeek, "-an", "-r", "1", "-vframes", "1", "-loglevel", "panic", dest}
	//args := []string{"-y", "-ss", timeToSeek, "-t", "1", "-loglevel", "panic", "-i", src, dest}
	//_, err = exec.Command("ffmpeg6", args...).Output()
	return
}

//ffmpeg6 -ss 00:02:23 -t 1 -i 'test.mkv' -loglevel panic -y x.jpeg
