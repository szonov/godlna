package backend

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/szonov/godlna/pkg/ffmpeg"
)

func thumbnailFile(videoFile string) string {
	return filepath.Dir(videoFile) + "/@eaDir/" + filepath.Base(videoFile) + "/SYNOVIDEO_VIDEO_SCREENSHOT.jpg"
}

func isThumbnailExists(videoFile string) bool {
	f := thumbnailFile(videoFile)
	if _, err := os.Stat(f); errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}

func makeThumbnail(videoFile string, duration int64, bookmark sql.NullInt64) error {
	var bm int64
	if bookmark.Valid {
		bm = bookmark.Int64
		// 0 - special case, video watched to 100% (Samsung TV send 0 to remove bookmark before jump to next file)
		if bm == 0 || bm > duration {
			bm = duration
		}
	}

	//slog.Info("MakeThumbnail", "dur", duration, "bm", bm, "bookmark", bookmark)

	return ffmpeg.Thumbnail(
		videoFile,
		thumbnailFile(videoFile),
		time.Duration(duration)*time.Millisecond,
		time.Duration(bm)*time.Millisecond,
		// --- then options ---
		ffmpeg.Width(480),
		ffmpeg.Height(300),
		ffmpeg.CompleteLeeway(5*time.Second),
		ffmpeg.JPEGQuality(80),
		ffmpeg.ProgressSize(20),
		//ffmpeg.ProgressPaddingX(30),
		//ffmpeg.ProgressPaddingY(145),
		ffmpeg.ProgressPositionBottom(),
	)
}
