package ffmpeg

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

var ffprobeBinPath = "ffprobe"

// SetFFProbeBinPath sets the global path to find and execute the `ffprobe` program
func SetFFProbeBinPath(binPath string) {
	ffprobeBinPath = binPath
}

type ProbeStream struct {
	CodecType  string `json:"codec_type"`
	CodecName  string `json:"codec_name"`
	SampleRate uint   `json:"sample_rate,string"`
	Channels   int    `json:"channels"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
}

func (s ProbeStream) Resolution() string {
	return fmt.Sprintf("%dx%d", s.Width, s.Height)
}

type ProbeFormat struct {
	FormatName      string  `json:"format_name"`
	DurationSeconds float64 `json:"duration,string"`
	Size            uint64  `json:"size,string"`
	BitRateOriginal uint    `json:"bit_rate,string"`
}

func (f ProbeFormat) Duration() time.Duration {
	return time.Duration(f.DurationSeconds * float64(time.Second))
}

func (f ProbeFormat) BitRate() uint {
	if f.BitRateOriginal > 8 {
		return f.BitRateOriginal / 8
	}
	return f.BitRateOriginal
}

type ProbeData struct {
	Format  ProbeFormat
	Streams []ProbeStream
}

func (d *ProbeData) firstStream(typ string) *ProbeStream {
	for _, stream := range d.Streams {
		if stream.CodecType == typ {
			return &stream
		}
	}
	return nil
}

func (d *ProbeData) FirstVideoStream() *ProbeStream {
	return d.firstStream("video")
}

func (d *ProbeData) FirstAudioStream() *ProbeStream {
	return d.firstStream("audio")
}

func Probe(src string) (data *ProbeData, err error) {
	data = &ProbeData{}
	args := []string{
		"-i", src, "-show_entries",
		"stream=index,codec_type,codec_name,sample_rate,channels,width,height : format=format_name,duration,size,bit_rate",
		"-of", "json", "-hide_banner", "-loglevel", "panic",
	}
	var b []byte
	b, err = exec.Command(ffprobeBinPath, args...).Output()
	if err != nil {
		return
	}
	err = json.Unmarshal(b, data)
	return
}
