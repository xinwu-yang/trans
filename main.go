package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"go.uber.org/zap"
)

const BufferSize = 32768

type Format struct {
	FileName       string  `json:"filename"`
	FormatLongName string  `json:"format_long_name"`
	Duration       float32 `json:"duration,string"`
	Size           int64   `json:"size,string"`
	BitRate        int64   `json:"bit_rate,string"`
}

type Stream struct {
	Index     int    `json:"index"`
	CodecType string `json:"codec_type"`
	CodecName string `json:"codec_name"`
}

type VideoSteam struct {
	Stream      Stream
	PixelFormat string `json:"pix_fmt"`
}

type AudioSteam struct {
	Stream   Stream
	Channels int `json:"channels,string"`
}

type MediaInfo struct {
	Format  Format   `json:"format"`
	Streams []Stream `json:"streams"`
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	baseDir := "H:\\INDAV"
	f, err := os.OpenFile(baseDir, os.O_RDONLY, os.ModeDir)
	if err != nil {
		sugar.Errorf("open dir has error", "err", err.Error())
	}
	defer f.Close()
	dirs, _ := f.ReadDir(-1)
	for _, dir := range dirs {
		if !dir.IsDir() {
			sugar.Infof("开始处理文件：", dir.Name())
			fileName := baseDir + "\\" + dir.Name()
			sugar.Infof("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", fileName)
			cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", fileName)
			ffprobeOut, _ := cmd.StdoutPipe()
			cmd.Start()
			var bt bytes.Buffer
			for {
				readData := make([]byte, BufferSize)
				i, _ := ffprobeOut.Read(readData)
				if i > 0 {
					bt.Write(readData[:i])
				} else {
					// 读取完输出后解析json
					videoInfoJson := bt.String()
					fmt.Println(videoInfoJson)
					format := Format{}
					//videoSteam := VideoSteam{}
					//audioSteam := AudioSteam{}
					mediaInfo := MediaInfo{}
					jsonBytes := bt.Bytes()
					var data map[string]interface{}
					err := json.Unmarshal(jsonBytes[:bt.Len()], &data)
					if err != nil {
						sugar.Errorf(err.Error())
						return
					}
					formatBytes, _ := json.Marshal(data["format"])
					json.Unmarshal(formatBytes, &format)
					handleVideoCodec, handleVideoPixFmt, handleAudioCodec := false, false, false
					streams := mediaInfo.Streams
					size := len(streams)
					// 根据参数判断是否处理视频
					if size > 0 {
						for i := 0; i < size; i++ {
							stream := streams[i]
							if stream.CodecType == "video" && stream.CodecName != "hevc" {
								handleVideoCodec = true
								continue
							}

							if stream.CodecType == "video" && stream.CodecName != "hevc" {
								handleVideoPixFmt = true
								continue
							}

							if stream.CodecType == "audio" && stream.CodecName != "aac" {
								handleAudioCodec = true
								continue
							}
						}
					}

					// 开始处理视频
					sugar.Infof("是否处理视频：", handleVideoCodec)
					if handleVideoCodec {
						handleVideo(fileName, mediaInfo, handleVideoCodec, handleVideoPixFmt, handleAudioCodec)
					}
					break
				}
			}
			ffprobeOut.Close()
		}
	}
}

/*处理视频*/
func handleVideo(fileName string, mediaInfo MediaInfo, handleVideoCodec bool, handleVideoPixFmt bool, handleAudioCodec bool) {
	//ffmpegCmdArray := []string{}
	//ffmpegCmd := exec.Command("ffmpeg", "-i", fileName)
}

func GetBytes(key interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(key)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
