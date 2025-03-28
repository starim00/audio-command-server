以下是更新后的 `README.md`，在安装 `whisper.cpp` 的步骤中添加了安装 `cmake` 以及相关依赖的详细说明，确保用户能够顺利完成环境搭建。

---

# AudioCommandServer

`AudioCommandServer` 是一个基于 Go 语言的 HTTP 服务，用于接收音频文件，将其转换为文字，并通过 DeepSeek API 提取特定指令（前进、左转、右转、后退）。项目集成了 `ffmpeg` 进行音频预处理和 `whisper.cpp` 进行本地语音转文字，以实现高效的音频处理。

## 功能
- **音频上传**：通过 HTTP POST 请求接收音频文件。
- **音频预处理**：使用 `ffmpeg` 将音频转换为 Whisper 所需的格式（WAV，16kHz，单声道）。
- **语音转文字**：使用 `whisper.cpp` 的 `tiny` 模型本地转录音频，默认支持中文。
- **指令提取**：调用 DeepSeek API 从转录文本中提取指令（前进、左转、右转、后退）。
- **JSON 响应**：返回处理结果或错误信息。

## 依赖项
- **Go**: 1.18 或更高版本
- **ffmpeg**: 用于音频格式转换
- **whisper.cpp**: 本地语音转文字模型
- **cmake**: 用于编译 `whisper.cpp`
- **DeepSeek API Key**: 用于指令提取

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

### 4. 安装 whisper.cpp
`whisper.cpp` 需要 `cmake` 和 C++ 编译器来构建。以下是完整步骤：

#### 4.1 安装 cmake
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

#### 4.2 安装 C++ 编译器
- **Ubuntu/Debian**:
  ```bash
  sudo apt install build-essential
  ```
- **macOS**: 安装 Xcode 命令行工具：
  ```bash
  xcode-select --install
  ```
- **Windows**: 安装 MinGW 或 Visual Studio（含 C++ 支持）。

#### 4.3 克隆并编译 whisper.cpp
```bash
git clone https://github.com/ggerganov/whisper.cpp
cd whisper.cpp
cmake -B build
cd build
make
```
编译完成后，可执行文件位于 `whisper.cpp/build/bin/whisper-cli`。

#### 4.4 下载模型
下载 `tiny` 模型：
```bash
cd ..
./models/download-ggml-model.sh tiny
```
确保 `whisper.cpp/build/bin/whisper-cli` 和 `whisper.cpp/models/ggml-tiny.bin` 位于项目根目录下，或在 `main.go` 中调整路径。

### 5. 配置 DeepSeek API Key
编辑 `main.go`，将 `apiKey` 常量替换为你的 DeepSeek API 密钥：
```go
const apiKey = "YOUR_API_KEY_HERE"
```

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
  {"command": "前进"}
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
- **API Key**：未设置 DeepSeek API Key 将导致 `401 Unauthorized` 错误。
- **音频格式**：支持任意音频格式，`ffmpeg` 会自动转换为 WAV（16kHz，单声道）。
- **性能**：使用 `tiny` 模型以提高速度，适合短音频处理。长音频或复杂场景可能需要更大模型（如 `base`）。
- **路径配置**：确保 `whisper-cli` 和模型文件路径与代码中一致。如果路径不同，修改 `main.go` 中的 `whisperBin` 和 `modelPath`。

## 扩展
- **动态模型**：通过请求参数支持切换 Whisper 模型。
- **多语言**：修改 `-l` 参数支持其他语言。
- **日志**：添加日志记录以便调试。