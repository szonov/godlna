package ffprobe

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var binPath = "ffprobe"

// SetBinPath sets the global path to find and execute the `ffprobe` program
func SetBinPath(path string) {
	binPath = path
}

// Autodetect try to find `ffprobe` program in predefined paths
func Autodetect() bool {
	lookup := []string{
		"/var/packages/ffmpeg7/target/bin/ffprobe",
		"/var/packages/ffmpeg6/target/bin/ffprobe",
		"ffprobe",
	}
	for _, p := range lookup {
		if val, err := exec.LookPath(p); err == nil {
			SetBinPath(val)
			return true
		}
	}
	return false
}

type Stream struct {
	CodecType  string `json:"codec_type"`
	CodecName  string `json:"codec_name"`
	SampleRate uint   `json:"sample_rate,string"`
	Channels   uint   `json:"channels"`
	Width      uint   `json:"width"`
	Height     uint   `json:"height"`
}

func (s Stream) Resolution() string {
	return fmt.Sprintf("%dx%d", s.Width, s.Height)
}

type Format struct {
	Name     string
	Duration time.Duration
	Size     uint
	BitRate  uint
}

type formatAlias struct {
	Name     string  `json:"format_name"`
	Duration float64 `json:"duration,string"`
	Size     uint    `json:"size,string"`
	BitRate  uint    `json:"bit_rate,string"`
}

func (m *Format) UnmarshalJSON(data []byte) error {
	var a formatAlias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	m.Name = a.Name
	m.Duration = time.Duration(a.Duration * float64(time.Second))
	m.Size = a.Size
	m.BitRate = a.BitRate
	if m.BitRate > 8 {
		m.BitRate /= 8
	}
	return nil
}

type Data struct {
	Format  Format
	Streams []Stream
}

func (d *Data) firstStream(typ string) *Stream {
	for _, stream := range d.Streams {
		if stream.CodecType == typ {
			return &stream
		}
	}
	return nil
}

func (d *Data) FirstVideoStream() *Stream {
	return d.firstStream("video")
}

func (d *Data) FirstAudioStream() *Stream {
	return d.firstStream("audio")
}

func Probe(src string) (data *Data, err error) {
	args := []string{
		"-i", src, "-show_entries",
		"stream=index,codec_type,codec_name,sample_rate,channels,width,height : format=format_name,duration,size,bit_rate",
		"-of", "json", "-hide_banner", "-loglevel", "panic",
	}
	var b []byte
	b, err = exec.Command(binPath, args...).Output()
	if err != nil {
		return
	}
	data = &Data{}
	if err = json.Unmarshal(b, data); err != nil {
		data = nil
	}
	return
}

func Duration(src string) (time.Duration, error) {
	cmd := exec.Command(binPath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		src)

	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("(ffprobe) can not get duration: %w", err)
	}

	durationStr := strings.TrimSpace(string(out))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("(ffprobe) invalid duration value: %w", err)
	}

	return time.Duration(duration * float64(time.Second)), nil
}
