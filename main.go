package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type deepSeekResponseBody struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type deepSeekMessage struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

type deepSeekRequestBody struct {
	Model            string            `json:"model"`
	Messages         []deepSeekMessage `json:"messages"`
	MaxTokens        int               `json:"max_tokens"`
	Temperature      float32           `json:"temperature"`
	TopP             int               `json:"top_p"`
	FrequencyPenalty int               `json:"frequency_penalty"`
	PresencePenalty  int               `json:"presence_penalty"`
}

type response struct {
	Command string `json:"command"`
	Error   string `json:"error,omitempty"`
}

const (
	deepSeekURL = "https://api.deepseek.com/v1/chat/completions" // 修正后的 URL
	modelName   = "deepseek-chat"
	apiKey      = "YOUR_API_KEY_HERE" // 请替换为你的 DeepSeek API Key
)

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

	command, err := callDeepseek(text)
	if err != nil {
		sendJSONError(w, fmt.Sprintf("文字转指令失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回 JSON 响应
	resp := response{Command: command}
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
	modelPath := "./whisper.cpp/models/ggml-tiny.bin"

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

func callDeepseek(commandText string) (string, error) {
	requestBody := deepSeekRequestBody{
		Model: modelName,
		Messages: []deepSeekMessage{
			{
				Content: "你将收到一段文本，你需要从里面提取出用户的指令，指令的类型分为：前进、左转、右转、后退。你需要直接回答指令的类型，不要增加其他内容",
				Role:    "system",
			},
			{
				Content: commandText,
				Role:    "user",
			},
		},
		MaxTokens:        2048,
		Temperature:      1.0,
		TopP:             1,
		FrequencyPenalty: 0,
		PresencePenalty:  0,
	}

	var requestData bytes.Buffer
	err := json.NewEncoder(&requestData).Encode(requestBody)
	if err != nil {
		return "", fmt.Errorf("编码请求体失败: %v", err)
	}

	req, err := http.NewRequest("POST", deepSeekURL, &requestData)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("DeepSeek API 返回错误: %d, %s", resp.StatusCode, string(body))
	}

	var deepResponse deepSeekResponseBody
	err = json.NewDecoder(resp.Body).Decode(&deepResponse)
	if err != nil {
		return "", fmt.Errorf("解码响应失败: %v", err)
	}

	if len(deepResponse.Choices) > 0 {
		return deepResponse.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("DeepSeek 未返回有效指令")
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
