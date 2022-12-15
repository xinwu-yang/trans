# h264-to-h265

### 功能

- 指定目录下视频转码`hev1`
- 指定目录下音频转码`aac`
- 像素格式转化为`yuv420p`
- 声道数量变更为`2`
- 支持GPU加速

### 如何使用

1. 下载[FFmpeg](https://www.gyan.dev/ffmpeg/builds/)

2. 配置把ffmpeg下`bin`目录添加到系统PATH环境变量

3. 使用h264-to-h265.exe

```
# 构建
go build

# 使用帮助
h264-to-h265.exe -h

Usage of h264-to-h265.exe:
  -d string
        路径，默认为当前执行目录
  -vc string
        视频编解码，默认 hevc_nvenc
  -r  bool
        是否递归子目录，默认 true

# 使用
h264-to-h265.exe -d "D:\\demo-video\\test"

# 启用GPU加速
h264-to-h265.exe -d "D:\\demo-video\\test" -vc hevc -r=false
```