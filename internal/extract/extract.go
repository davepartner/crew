package extract

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Text returns readable text for a stored upload. It supports text-like files,
// PDFs (embedded text, then OCR for scans) and images (Tesseract).
func Text(ctx context.Context, path, mime string, raw []byte) string {
	if strings.HasPrefix(mime, "text/") || strings.Contains(mime, "json") || strings.Contains(mime, "csv") {
		return clip(string(raw))
	}

	switch {
	case strings.Contains(mime, "pdf"):
		if text := pdfText(ctx, path); len(strings.TrimSpace(text)) >= 30 {
			return clip(text)
		}
		return clip(pdfOCR(ctx, path))
	case strings.HasPrefix(mime, "image/"):
		return clip(imageOCR(ctx, path))
	}

	return ""
}

func pdfText(ctx context.Context, path string) string {
	command := exec.CommandContext(ctx, "pdftotext", "-layout", path, "-")
	command.WaitDelay = 3 * time.Second
	out, err := command.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func pdfOCR(ctx context.Context, path string) string {
	directory, err := os.MkdirTemp("", "crew-pdf-")
	if err != nil {
		return ""
	}
	defer os.RemoveAll(directory)

	prefix := filepath.Join(directory, "page")
	command := exec.CommandContext(ctx, "pdftoppm", "-png", "-r", "150", "-f", "1", "-l", "5", path, prefix)
	command.WaitDelay = 3 * time.Second
	if err := command.Run(); err != nil {
		return ""
	}

	pages, _ := filepath.Glob(prefix + "-*.png")
	var builder strings.Builder
	for _, page := range pages {
		builder.WriteString(imageOCR(ctx, page))
		builder.WriteString("\n")
	}
	return builder.String()
}

func imageOCR(ctx context.Context, path string) string {
	command := exec.CommandContext(ctx, "tesseract", path, "stdout", "-l", "eng")
	command.WaitDelay = 3 * time.Second
	var stderr bytes.Buffer
	command.Stderr = &stderr
	out, err := command.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func clip(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 200000 {
		value = value[:200000]
	}
	return value
}

func Available() error {
	for _, tool := range []string{"pdftotext", "tesseract"} {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("%s is not installed", tool)
		}
	}
	return nil
}

func TextWithKey(ctx context.Context, path, mime string, raw []byte, ocrKey string) string {
	text := Text(ctx, path, mime, raw)
	if strings.TrimSpace(ocrKey) == "" {
		return text
	}
	if !strings.Contains(mime, "pdf") && !strings.HasPrefix(mime, "image/") {
		return text
	}

	// A configured OCR key is also used to cross-check local OCR: keep
	// whichever result contains more text.
	remote := ocrSpace(ctx, path, ocrKey)
	if len(strings.TrimSpace(remote)) > len(strings.TrimSpace(text)) {
		return remote
	}
	return text
}
