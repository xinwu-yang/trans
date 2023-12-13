# Transcoding tool

### 功能

- 指定目录下视频转码`hev1`
- 指定目录下音频转码`aac`
- 像素格式转化为`yuv420p`
- 声道数量变更为`2`
- 可指定`CRF`值
- 支持GPU加速
- 支持递归目录

### 如何使用

1. 下载[FFmpeg](https://www.gyan.dev/ffmpeg/builds/)

2. 配置把ffmpeg下`bin`目录添加到系统PATH环境变量

3. 使用trans.exe

```
# 构建
go build

# 使用帮助
trans.exe -h

Usage of trans.exe:
  -crf string
        视频压缩质量 (default "28")
  -d string
        视频路径 (default "./")
  -r    是否递归子目录（useage: -r=false） (default true)
  -vc string
        视频编码 (default "hevc_nvenc")

# 基本使用
trans.exe -d "D:\\demo-video\\test"

# 使用CPU转码
trans.exe -d "D:\\demo-video\\test" -vc hevc
```

> 文件名称带有 NOT-HANDLE 则会跳过处理