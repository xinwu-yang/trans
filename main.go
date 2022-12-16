package main

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const FileSeparator = string(os.PathSeparator)
const Version = "1.1.1"

// 全局日志
var sugar *zap.SugaredLogger
var crf string
var videoCodec string
var recursive bool

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
	encoderConfig.EncodeTime = func(t time.Time, pae zapcore.PrimitiveArrayEncoder) {
		pae.AppendString(t.Format("2006-01-02 15:04:05"))
	}
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
	flag.StringVar(&path, "d", "./", "视频路径（default: ./）")
	flag.StringVar(&videoCodec, "vc", "hevc_nvenc", "视频编码（default: hevc_nvenc）")
	flag.StringVar(&crf, "crf", "28", "视频压缩质量（default: 28）")
	flag.BoolVar(&recursive, "r", true, "是否递归子目录（default: true）（useage: -r=false）")
	// 解析注册的 flag
	flag.Parse()

	// 配置日志
	core := zapcore.NewCore(getEncoder(), zapcore.AddSync(os.Stdout), zapcore.DebugLevel)
	logger := zap.New(core, zap.AddCaller())
	defer logger.Sync()
	sugar = logger.Sugar()

	// 打印版本号
	sugar.Infof("H264-To-H265 Version: %s", Version)

	// 获取绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		sugar.Error(err.Error())
	}
	readFiles(absPath)
}

/* 读取目录文件列表 */
func readFiles(path string) {
	dirs, _ := os.ReadDir(path)
	dirSize := len(dirs)
	sugar.Info("--------------------------切换目录--------------------------")
	sugar.Infof("当前处理目录：%s", path)
	sugar.Infof("目录下文件和子目录总数：%v", dirSize)
	for _, dir := range dirs {
		dirName := dir.Name()
		if !dir.IsDir() {
			execFFprobeCmd(dirName, path)
		} else {
			if recursive {
				childDirPath := path + FileSeparator + dirName
				readFiles(childDirPath)
			}
		}
	}
}

/* 获取视频编码信息 */
func execFFprobeCmd(fileName string, path string) {
	sugar.Info("--------------------------文件处理--------------------------")
	sugar.Infof("开始处理文件：%s", fileName)
	sugar.Infof("CMD: ffprobe -v quiet -print_format json -show_format -show_streams %v", path+FileSeparator+fileName)
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", path+FileSeparator+fileName)
	ffprobeOut, _ := cmd.StdoutPipe()
	cmd.Start()
	jsonBytes, _ := io.ReadAll(ffprobeOut)
	// 读取完输出后解析json
	format := Format{}
	videoSteam := VideoSteam{}
	audioSteam := AudioSteam{}
	mediaInfo := MediaInfo{}
	var data map[string]interface{}
	err := json.Unmarshal(jsonBytes, &data)
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
		execFFmpegCmd(fileName, path, handleVideoCodec, handleVideoPixFmt, handleAudioCodec, handleAudioChannels)
	}
	ffprobeOut.Close()
}

/* 处理视频 */
func execFFmpegCmd(fileName string, path string, handleVideoCodec bool, handleVideoPixFmt bool, handleAudioCodec bool, handleAudioChannels bool) {
	ffmpegCmdArray := []string{"-i", path + FileSeparator + fileName}
	ffmpegCmdArray = append(ffmpegCmdArray, "-crf", crf)
	if handleVideoCodec {
		ffmpegCmdArray = append(ffmpegCmdArray, "-c:v", videoCodec)
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
	tempName := fileName[:strings.LastIndexAny(fileName, ".")]
	ffmpegCmdArray = append(ffmpegCmdArray, tempName+"-HEVC.mp4")
	sugar.Infof("CMD: ffmpeg %v", ffmpegCmdArray)
	ffmpegCmd := exec.Command("ffmpeg", ffmpegCmdArray...)
	ffmpegCmd.Dir = path
	ffmpegCmd.Stdout = os.Stdout
	ffmpegCmd.Stderr = os.Stderr
	if err := ffmpegCmd.Run(); err != nil {
		sugar.Error(err.Error())
	}
}
