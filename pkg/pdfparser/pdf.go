package pdfparser

import (
	"bytes"
	"fmt"

	"github.com/ledongthuc/pdf"
)

// ExtractTextFromPDF 极简、纯粹的解析函数，只依赖字节流
func ExtractTextFromPDF(fileData []byte) (string, error) {
	// 1. 直接获取文件大小，不再需要外部传参
	fileSize := int64(len(fileData))
	// 2. 转换为 io.ReaderAt
	reader := bytes.NewReader(fileData)

	// 初始化 PDF reader
	f, err := pdf.NewReader(reader, fileSize)
	if err != nil {
		return "", fmt.Errorf("pdf reader init failed: %w", err)
	}

	var textBuilder bytes.Buffer
	totalPage := f.NumPage()

	// 逐页读取纯文本
	for pageIndex := 1; pageIndex <= totalPage; pageIndex++ {
		p := f.Page(pageIndex)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			return "", fmt.Errorf("failed to read page %d: %w", pageIndex, err)
		}
		textBuilder.WriteString(text)
		textBuilder.WriteString("\n") // 每页加个换行
	}

	return textBuilder.String(), nil
}
