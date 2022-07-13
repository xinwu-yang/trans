package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
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

type MediaInfo struct {
	Format  Format   `json:"format"`
	Streams []Stream `json:"streams"`
}

func main() {
	baseDir := "H:\\INDAV"
	f, err := os.OpenFile(baseDir, os.O_RDONLY, os.ModeDir)
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer f.Close()
	dirs, _ := f.ReadDir(-1)
	for _, dir := range dirs {
		if !dir.IsDir() {
			log.Println("开始处理文件：" + dir.Name())
			log.Println("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", baseDir+"\\"+dir.Name())
			cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", baseDir+"\\"+dir.Name())
			ffmpegOut, _ := cmd.StdoutPipe()
			cmd.Start()
			var bt bytes.Buffer
			for {
				readData := make([]byte, BufferSize)
				i, _ := ffmpegOut.Read(readData)
				if i > 0 {
					bt.Write(readData[:i])
				} else {
					// 读取完输出后解析json
					videoInfoJson := bt.String()
					log.Println(videoInfoJson)
					mediaInfo := MediaInfo{}
					jsonBytes := bt.Bytes()
					err := json.Unmarshal(jsonBytes[:bt.Len()], &mediaInfo)
					if err != nil {
						log.Panicln(err.Error())
					}
					log.Println(mediaInfo.Format.FileName)
					break
				}
			}
			ffmpegOut.Close()
			fmt.Println()
		}
	}
}
