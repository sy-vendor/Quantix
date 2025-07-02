package analysis

import (
	"os"
	"os/exec"
)

type ExportFormat string

// 导出分析报告为 Markdown、HTML 或 PDF
func ExportReport(result AnalysisResult, format string) (string, error) {
	var content string
	filename := result.SavedFile
	base := filename[:len(filename)-5] // 去掉 .json
	switch format {
	case "md":
		content = "# AI 智能分析报告\n\n" + result.Report
		filename = base + ".md"
		path := "history/" + filename
		err := os.WriteFile(path, []byte(content), 0644)
		return filename, err
	case "html":
		content = "<html><head><meta charset=\"utf-8\"></head><body><pre>" + result.Report + "</pre></body></html>"
		filename = base + ".html"
		path := "history/" + filename
		err := os.WriteFile(path, []byte(content), 0644)
		return filename, err
	case "pdf":
		// 先生成 HTML
		htmlFile := "history/" + base + ".html"
		pdfFile := "history/" + base + ".pdf"
		content = "<html><head><meta charset=\"utf-8\"></head><body><pre>" + result.Report + "</pre></body></html>"
		_ = os.WriteFile(htmlFile, []byte(content), 0644)
		// 调用 wkhtmltopdf
		cmd := exec.Command("wkhtmltopdf", htmlFile, pdfFile)
		err := cmd.Run()
		if err != nil {
			return "", err
		}
		return base + ".pdf", nil
	default:
		return "", nil
	}
}
