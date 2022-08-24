package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/thinkeridea/go-extend/exunicode/exutf8"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const BufferSize = 4096

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
	Stream
	PixelFormat string `json:"pix_fmt"`
}

type AudioSteam struct {
	Stream
	Channels int `json:"channels"`
}

type MediaInfo struct {
	Format     Format
	VideoSteam VideoSteam
	AudioSteam AudioSteam
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	// 时间函数可以自定义
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// 输出到文件
// func getLogWriter() zapcore.WriteSyncer {
// 	file, _ := os.Create("./app.log")
// 	return zapcore.AddSync(file)
// }

func main() {
	// 定义几个变量，用于接收命令行的参数值
	var path string
	var videoCodec string
	flag.StringVar(&path, "d", "", "路径，默认为空")
	flag.StringVar(&videoCodec, "vc", "hevc", "视频编解码")
	// 解析注册的 flag
	flag.Parse()

	core := zapcore.NewCore(getEncoder(), zapcore.AddSync(os.Stdout), zapcore.DebugLevel)
	logger := zap.New(core, zap.AddCaller())
	defer logger.Sync()
	sugar := logger.Sugar()

	sugar.Infof("系统：%s", runtime.GOOS)
	sugar.Infof("架构：%s", runtime.GOARCH)

	//baseDir := "D:\\demo-video"
	f, err := os.OpenFile(path, os.O_RDONLY, os.ModeDir)
	if err != nil {
		sugar.Errorf("open dir has error", "err", err.Error())
	}
	defer f.Close()
	dirs, _ := f.ReadDir(-1)
	for _, dir := range dirs {
		if !dir.IsDir() {
			sugar.Infof("开始处理文件：%s", dir.Name())
			fileName := path + "\\" + dir.Name()
			sugar.Infof("CMD: ffprobe -v quiet -print_format json -show_format -show_streams %v", fileName)
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
					format := Format{}
					videoSteam := VideoSteam{}
					audioSteam := AudioSteam{}
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

					streams := data["streams"]
					streamsBytes, _ := json.Marshal(streams)
					var streamData []map[string]interface{}
					json.Unmarshal(streamsBytes, &streamData)

					for _, stream := range streamData {
						streamBytes, _ := json.Marshal(stream)
						if stream["codec_type"] == "video" {
							json.Unmarshal(streamBytes, &videoSteam)
						} else if stream["codec_type"] == "audio" {
							json.Unmarshal(streamBytes, &audioSteam)
						}
					}
					mediaInfo.Format = format
					mediaInfo.VideoSteam = videoSteam
					mediaInfo.AudioSteam = audioSteam
					handleVideoCodec, handleVideoPixFmt, handleAudioCodec, handleAudioChannels := false, false, false, false

					// 根据参数判断是否处理视频
					if videoSteam.CodecType == "video" && videoSteam.CodecName != "hevc" {
						handleVideoCodec = true
					}
					if videoSteam.CodecType == "video" && videoSteam.PixelFormat != "yuv420p" {
						handleVideoPixFmt = true
					}
					if audioSteam.CodecType == "audio" && audioSteam.CodecName != "aac" {
						handleAudioCodec = true
					}
					if audioSteam.CodecType == "audio" && audioSteam.Channels > 2 {
						handleAudioChannels = true
					}
					// 开始处理视频
					sugar.Infof("是否处理视频编码：%v", handleVideoCodec)
					sugar.Infof("是否处理视频像素格式：%v", handleVideoPixFmt)
					sugar.Infof("是否处理音频编码：%v", handleAudioCodec)
					sugar.Infof("是否处理音频声道数：%v", handleAudioChannels)
					if handleVideoCodec || handleVideoPixFmt || handleAudioCodec || handleAudioChannels {
						handleVideo(fileName, dir, path, videoCodec, sugar, handleVideoCodec, handleVideoPixFmt, handleAudioCodec, handleAudioChannels)
					}
					break
				}
			}
			ffprobeOut.Close()
		}
	}
}

/*处理视频*/
func handleVideo(fileName string, dir fs.DirEntry, path string, vc string, sugar *zap.SugaredLogger, handleVideoCodec bool, handleVideoPixFmt bool, handleAudioCodec bool, handleAudioChannels bool) {
	ffmpegCmdArray := []string{"-i", fileName}

	if handleVideoCodec {
		ffmpegCmdArray = append(ffmpegCmdArray, "-c:v", vc)
	}

	if handleVideoPixFmt {
		ffmpegCmdArray = append(ffmpegCmdArray, "-pix_fmt", "yuv420p")
	}

	if handleAudioCodec {
		ffmpegCmdArray = append(ffmpegCmdArray, "-c:a", "aac")
	}

	if handleAudioChannels {
		ffmpegCmdArray = append(ffmpegCmdArray, "-ac", "2")
	}

	tempName := exutf8.RuneSubString(dir.Name(), 0, strings.LastIndexAny(dir.Name(), "."))
	ffmpegCmdArray = append(ffmpegCmdArray, tempName+"-HEVC.mp4")
	sugar.Infof("", ffmpegCmdArray)
	ffmpegCmd := exec.Command("ffmpeg", ffmpegCmdArray...)
	ffmpegCmd.Dir = path
	ffmpegOut, _ := ffmpegCmd.StdoutPipe()

	if err := ffmpegCmd.Start(); err != nil { // 运行命令
		fmt.Println(err.Error())
	}

	readData := make([]byte, BufferSize)
	i, _ := ffmpegOut.Read(readData)

	for {
		if i > 0 {
			fmt.Println(readData)
		} else {
			break
		}
	}

	defer ffmpegOut.Close()
}
