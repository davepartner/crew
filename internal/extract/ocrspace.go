package extract

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func ocrSpace(ctx context.Context, path, key string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("apikey", key); err != nil {
		return ""
	}
	_ = writer.WriteField("language", "eng")
	_ = writer.WriteField("isOverlayRequired", "false")
	_ = writer.WriteField("OCREngine", "2")

	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return ""
	}
	if _, err := io.Copy(part, io.LimitReader(file, 10<<20)); err != nil {
		return ""
	}
	if err := writer.Close(); err != nil {
		return ""
	}

	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.ocr.space/parse/image", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	response, err := (&http.Client{Timeout: 45 * time.Second}).Do(request)
	if err != nil {
		return ""
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ""
	}

	var payload struct {
		ParsedResults []struct {
			ParsedText string `json:"ParsedText"`
		} `json:"ParsedResults"`
		IsErroredOnProcessing bool     `json:"IsErroredOnProcessing"`
		ErrorMessage          []string `json:"ErrorMessage"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&payload); err != nil || payload.IsErroredOnProcessing {
		return ""
	}
	var builder strings.Builder
	for _, result := range payload.ParsedResults {
		builder.WriteString(result.ParsedText)
	}
	return builder.String()
}
