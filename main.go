package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/liuzl/gocc"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type response struct {
	Text  string `json:"text"`
	Error string `json:"error,omitempty"`
}

func uploadAndTranscribeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONError(w, "只支持 POST 方法", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		sendJSONError(w, "无法解析表单数据", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("audio")
	if err != nil {
		sendJSONError(w, "无法获取音频文件", http.StatusBadRequest)
		return
	}
	defer file.Close()

	err = os.MkdirAll("uploads", 0755)
	if err != nil {
		sendJSONError(w, "无法创建目录", http.StatusInternalServerError)
		return
	}

	originalFilePath := filepath.Join("uploads", "original_"+handler.Filename)
	originalFile, err := os.Create(originalFilePath)
	if err != nil {
		sendJSONError(w, "无法创建原始文件", http.StatusInternalServerError)
		return
	}
	defer originalFile.Close()

	_, err = io.Copy(originalFile, file)
	if err != nil {
		sendJSONError(w, "原始文件保存失败", http.StatusInternalServerError)
		return
	}

	convertedFilePath := filepath.Join("uploads", "converted_"+handler.Filename+".wav")
	err = convertAudioWithFFmpeg(originalFilePath, convertedFilePath)
	if err != nil {
		sendJSONError(w, fmt.Sprintf("音频转换失败: %v", err), http.StatusInternalServerError)
		return
	}

	text, err := transcribeWithWhisper(convertedFilePath)
	if err != nil {
		sendJSONError(w, fmt.Sprintf("语音转文字失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 清理 Whisper 输出，去除多余换行符和空白
	text = strings.TrimSpace(text)

	// 返回 JSON 响应
	text, err = toSimplifiedChinese(text)
	if err != nil {
		sendJSONError(w, fmt.Sprintf("繁简处理失败: %v", err), http.StatusInternalServerError)
		return
	}
	resp := response{Text: text}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	// 清理临时文件
	os.Remove(originalFilePath)
	os.Remove(convertedFilePath)
}

func convertAudioWithFFmpeg(inputPath, outputPath string) error {
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-ar", "16000", "-ac", "1", "-y", outputPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("ffmpeg 转换失败: %v, stderr: %s", err, stderr.String())
	}
	return nil
}

func transcribeWithWhisper(filePath string) (string, error) {
	whisperBin := "./whisper.cpp/build/bin/whisper-cli" // 更新为你的 whisper 可执行文件路径
	modelPath := "./whisper.cpp/models/ggml-medium.bin"

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

// toSimplifiedChinese 使用 opencc-go 转换为简体中文
func toSimplifiedChinese(text string) (string, error) {
	// 初始化 OpenCC，t2s 表示繁体到简体
	converter, err := gocc.New("t2s")
	if err != nil {
		return "", fmt.Errorf("初始化 OpenCC 失败: %v", err)
	}
	// 进行转换
	converted, err := converter.Convert(text)
	if err != nil {
		return "", fmt.Errorf("繁简转换失败: %v", err)
	}
	return converted, nil
}

func sendJSONError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response{Error: msg})
}

func main() {
	http.HandleFunc("/transcribe", uploadAndTranscribeHandler)
	fmt.Println("服务器启动在 :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("服务器启动失败:", err)
	}
}
