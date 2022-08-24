# h264-to-h265

### 功能

- 指定目录下视频转码`hev1`
- 指定目录下音频转码`aac`
- 像素格式转化为`yuv420p`
- 声道数量变更为`2`
- 支持GPU加速

### 如何使用

```
# 构建
go build

# 使用帮助
h264-to-h265.exe -h

# 使用
h264-to-h265.exe -d "D:\\demo-video\\test"

# GPU加速
h264-to-h265.exe -d "D:\\demo-video\\test" -vc hevc_nvenc
```