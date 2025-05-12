# AudioCommandServer

`AudioCommandServer` 是一个基于 Go 语言的 HTTP 服务，用于接收音频文件并将其转换为简体中文文本。项目集成了 `ffmpeg` 进行音频预处理、`whisper.cpp` 进行本地语音转文字，以及 `gocc`（基于 OpenCC）进行繁简中文转换。

## 功能
- **音频上传**：通过 HTTP POST 请求接收音频文件。
- **音频预处理**：使用 `ffmpeg` 将音频转换为 Whisper 所需的格式（WAV，16kHz，单声道）。
- **语音转文字**：使用 `whisper.cpp` 的 `medium` 模型本地转录音频，默认支持中文。
- **简体中文转换**：使用 `gocc` 确保转录结果为简体中文。
- **JSON 响应**：返回转录文本或错误信息。

## 依赖项
- **Go**: 1.18 或更高版本
- **ffmpeg**: 用于音频格式转换
- **whisper.cpp**: 本地语音转文字模型
- **cmake**: 用于编译 `whisper.cpp`
- **libopencc**: 用于繁简中文转换

## 安装

### 1. 克隆项目
```bash
git clone https://github.com/yourusername/AudioCommandServer.git
cd AudioCommandServer
```

### 2. 安装 Go 依赖
确保 Go 环境已配置好，然后运行：
```bash
go mod init AudioCommandServer  # 如果尚未初始化
go get github.com/liuzl/gocc
go mod tidy
```

### 3. 安装 ffmpeg
根据操作系统安装 `ffmpeg`：
- **Ubuntu/Debian**:
  ```bash
  sudo apt update
  sudo apt install ffmpeg
  ```
- **macOS**:
  ```bash
  brew install ffmpeg
  ```
- **Windows**: 从 [ffmpeg 官网](https://ffmpeg.org/download.html) 下载并添加到 PATH。

### 4. 安装 libopencc
`gocc` 需要底层的 OpenCC 库：
- **Ubuntu/Debian**:
  ```bash
  sudo apt install libopencc-dev
  ```
- **macOS**:
  ```bash
  brew install opencc
  ```
- **Windows**: 下载 OpenCC 预编译库（见 [OpenCC GitHub](https://github.com/BYVoid/OpenCC)）并配置 PATH。

### 5. 安装 whisper.cpp
`whisper.cpp` 需要 `cmake` 和 C++ 编译器来构建。

#### 5.1 安装 cmake
根据操作系统安装 `cmake`：
- **Ubuntu/Debian**:
  ```bash
  sudo apt update
  sudo apt install cmake
  ```
- **macOS**:
  ```bash
  brew install cmake
  ```
- **Windows**: 从 [CMake 官网](https://cmake.org/download/) 下载安装包，安装时选择“添加到系统 PATH”。

验证安装：
```bash
cmake --version
```

#### 5.2 安装 C++ 编译器
- **Ubuntu/Debian**:
  ```bash
  sudo apt install build-essential
  ```
- **macOS**: 安装 Xcode 命令行工具：
  ```bash
  xcode-select --install
  ```
- **Windows**: 安装 MinGW 或 Visual Studio（含 C++ 支持）。

#### 5.3 克隆并编译 whisper.cpp
```bash
git clone https://github.com/ggerganov/whisper.cpp
cd whisper.cpp
cmake -B build
cd build
make
```
编译完成后，可执行文件位于 `whisper.cpp/build/bin/whisper-cli`.

#### 5.4 下载模型
下载 `medium` 模型以提高中文识别精度：
```bash
cd ..
./models/download-ggml-model.sh medium
```
确保 `whisper.cpp/build/bin/whisper-cli` 和 `whisper.cpp/models/ggml-medium.bin` 位于项目根目录下，或在 `main.go` 中调整路径。

## 使用方法

### 1. 运行服务
```bash
go run main.go
```
服务将在 `http://localhost:8080` 启动。

### 2. 上传音频文件
使用 `curl` 或其他 HTTP 客户端发送 POST 请求：
```bash
curl -X POST -F "audio=@/path/to/audio.mp3" http://localhost:8080/transcribe
```

### 3. 响应示例
- **成功**：
  ```json
  {"text": "请前进十米"}
  ```
- **失败**：
  ```json
  {"error": "无法获取音频文件"}
  ```

## 项目结构
```
AudioCommandServer/
├── main.go           # 主程序文件
├── uploads/          # 临时存储上传和转换后的音频文件
├── whisper.cpp/      # whisper.cpp 源码和模型（需手动安装）
└── README.md         # 项目说明
```

## 注意事项
- **音频格式**：支持任意音频格式，`ffmpeg` 会自动转换为 WAV（16kHz，单声道）。
- **简体中文**：使用 `gocc` 确保转录结果为简体中文，`-l zh` 参数优先支持中文。
- **性能**：使用 `medium` 模型平衡精度和速度。长音频或复杂场景可考虑 `large` 模型。
- **路径配置**：确保 `whisper-cli` 和模型文件路径与代码中一致。如果路径不同，修改 `main.go` 中的 `whisperBin` 和 `modelPath`。
- **OpenCC**：确保 `libopencc` 已正确安装，否则编译可能失败。

## 扩展
- **动态模型**：通过请求参数支持切换 Whisper 模型。
- **多语言**：修改 `-l` 参数支持其他语言。
- **日志**：添加日志记录以便调试。