package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

func uploadAndTranscribeHandler(w http.ResponseWriter, r *http.Request) {
	// 检查请求方法
	if r.Method != http.MethodPost {
		http.Error(w, "只支持 POST 方法", http.StatusMethodNotAllowed)
		return
	}

	// 解析表单，限制内存为 10MB
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "无法解析表单数据", http.StatusBadRequest)
		return
	}

	// 获取音频文件
	file, handler, err := r.FormFile("audio")
	if err != nil {
		http.Error(w, "无法获取音频文件", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 创建 uploads 目录
	err = os.MkdirAll("uploads", 0755)
	if err != nil {
		http.Error(w, "无法创建目录", http.StatusInternalServerError)
		return
	}

	// 保存原始音频文件
	originalFilePath := filepath.Join("uploads", "original_"+handler.Filename)
	originalFile, err := os.Create(originalFilePath)
	if err != nil {
		http.Error(w, "无法创建原始文件", http.StatusInternalServerError)
		return
	}
	defer originalFile.Close()

	_, err = io.Copy(originalFile, file)
	if err != nil {
		http.Error(w, "原始文件保存失败", http.StatusInternalServerError)
		return
	}

	// 使用 ffmpeg 转换为 Whisper 所需的格式
	convertedFilePath := filepath.Join("uploads", "converted_"+handler.Filename+".wav")
	err = convertAudioWithFFmpeg(originalFilePath, convertedFilePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("音频转换失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 调用 Whisper 进行转录
	text, err := transcribeWithWhisper(convertedFilePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("语音转文字失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回转录结果
	fmt.Fprintf(w, "音频文件 %s 的转录结果:\n%s\n", handler.Filename, text)

	// 可选：清理临时文件
	os.Remove(originalFilePath)
	os.Remove(convertedFilePath)
}

// convertAudioWithFFmpeg 使用 ffmpeg 转换音频格式
func convertAudioWithFFmpeg(inputPath, outputPath string) error {
	// ffmpeg -i input -ar 16000 -ac 1 output.wav
	// -ar 16000: 采样率 16kHz
	// -ac 1: 单声道
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-ar", "16000", "-ac", "1", "-y", outputPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("ffmpeg 转换失败: %v, stderr: %s", err, stderr.String())
	}
	return nil
}

// transcribeWithWhisper 调用 whisper.cpp 的 main 程序进行转录
func transcribeWithWhisper(filePath string) (string, error) {
	whisperBin := "./whisper.cpp/build/bin/whisper-cli" // whisper.cpp 的 main 可执行文件路径
	modelPath := "./whisper.cpp/models/ggml-tiny.bin"   // 模型文件路径

	// 执行命令：./main -m ggml-base.bin -f audio.wav -l zh
	cmd := exec.Command(whisperBin, "-m", modelPath, "-f", filePath, "-l", "zh")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Whisper 执行失败: %v, stderr: %s", err, stderr.String())
	}

	return out.String(), nil
}

func main() {
	// 注册路由
	http.HandleFunc("/transcribe", uploadAndTranscribeHandler)

	// 启动服务器
	fmt.Println("服务器启动在 :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("服务器启动失败:", err)
	}
}
