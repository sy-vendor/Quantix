package analysis

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"encoding/json"
	"net/http"
	"strconv"

	"context"

	"regexp"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/mattn/go-runewidth"
	"github.com/russross/blackfriday/v2"
	"golang.org/x/term"
)

// 类型定义补充

// AnalysisParams 用于传递分析参数
// 可根据main.go和ai.go的实际用法调整字段
// StockData、TechnicalIndicator用于chart.go

type AnalysisParams struct {
	APIKey       string
	Model        string
	StockCodes   []string
	Start        string
	End          string
	SearchMode   bool
	HybridSearch bool // 新增，混合模式
	Periods      []string
	Dims         []string
	Output       []string
	Confidence   bool
	Risk         string
	Scope        []string
	Lang         string
	Prompt       string // 可选，手动传递prompt
}

type AnalysisResult struct {
	StockCode string
	Report    string
	SavedFile string
	Err       error
}

type StockData struct {
	Date   time.Time
	Open   float64
	Close  float64
	Low    float64
	High   float64
	Volume float64
}

type TechnicalIndicator struct {
	MA5  float64
	MA10 float64
	MA20 float64
	MA60 float64
}

// 函数声明补充
func FetchStockHistory(stockCode, start, end, apiKey string) ([]StockData, []TechnicalIndicator, error) {
	// 腾讯API symbol格式：sh600036、sz000001
	symbol := stockCode
	if len(stockCode) == 6 && stockCode[0] == '6' {
		symbol = "sh" + stockCode
	} else if len(stockCode) == 6 && (stockCode[0] == '0' || stockCode[0] == '3') {
		symbol = "sz" + stockCode
	}
	url := "https://web.ifzq.gtimg.cn/appstock/app/kline/kline?param=" + symbol + ",day,,,320"
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var data struct {
		Data map[string]struct {
			Day [][]interface{} `json:"day"`
		} `json:"data"`
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, nil, err
	}
	var stockData []StockData
	var closes []float64
	for _, v := range data.Data {
		for _, item := range v.Day {
			// item: [日期,开,收,高,低,成交量,...]
			if len(item) < 6 {
				continue
			}
			dateStr := item[0].(string)
			dt, _ := time.Parse("2006-01-02", dateStr)
			open, _ := strconv.ParseFloat(item[1].(string), 64)
			close, _ := strconv.ParseFloat(item[2].(string), 64)
			high, _ := strconv.ParseFloat(item[3].(string), 64)
			low, _ := strconv.ParseFloat(item[4].(string), 64)
			vol, _ := strconv.ParseFloat(item[5].(string), 64)
			stockData = append(stockData, StockData{
				Date:   dt,
				Open:   open,
				Close:  close,
				High:   high,
				Low:    low,
				Volume: vol,
			})
			closes = append(closes, close)
		}
	}
	// 本地计算MA5/10/20/60
	ma := func(arr []float64, n int, idx int) float64 {
		if idx+1 < n {
			return 0
		}
		sum := 0.0
		for i := idx + 1 - n; i <= idx; i++ {
			sum += arr[i]
		}
		return sum / float64(n)
	}
	var indicators []TechnicalIndicator
	for i := range stockData {
		indicators = append(indicators, TechnicalIndicator{
			MA5:  ma(closes, 5, i),
			MA10: ma(closes, 10, i),
			MA20: ma(closes, 20, i),
			MA60: ma(closes, 60, i),
		})
	}
	return stockData, indicators, nil
}

func BuildPrompt(params AnalysisParams) string {
	// 构建分析提示词
	prompt := fmt.Sprintf("请对股票代码 %s 进行智能分析。\n", strings.Join(params.StockCodes, ","))
	prompt += fmt.Sprintf("分析时间范围：%s 至 %s\n", params.Start, params.End)
	if len(params.Periods) > 0 {
		prompt += fmt.Sprintf("预测周期：%s\n", strings.Join(params.Periods, "、"))
	}
	if len(params.Dims) > 0 {
		prompt += fmt.Sprintf("分析维度：%s\n", strings.Join(params.Dims, "、"))
	}
	if params.Risk != "" {
		prompt += fmt.Sprintf("风险偏好：%s\n", params.Risk)
	}
	if params.Lang != "" {
		prompt += fmt.Sprintf("输出语言：%s\n", params.Lang)
	}
	prompt += "\n请提供详细的技术分析和投资建议。"
	return prompt
}

func markdownToHTML(md string) string {
	html := blackfriday.Run([]byte(md))
	return string(html)
}

func replaceImagesWithAbsHTML(md string) string {
	imgRe := regexp.MustCompile(`!\[.*?\]\((.*?)\)`)
	return imgRe.ReplaceAllStringFunc(md, func(s string) string {
		m := imgRe.FindStringSubmatch(s)
		if len(m) < 2 {
			return s
		}
		imgPath := m[1]
		abs, err := filepath.Abs(imgPath)
		if err != nil {
			return s
		}
		return fmt.Sprintf(`<img src="file://%s" style="max-width:100%%;">`, abs)
	})
}

// 新增：将行情数据结构化为表格文本
func FormatStockDataTable(stockData []StockData, indicators []TechnicalIndicator) string {
	if len(stockData) == 0 {
		return ""
	}
	head := "\n【历史行情数据表】\n| 日期 | 开盘 | 收盘 | 最高 | 最低 | 成交量 | MA5 | MA10 | MA20 | MA60 |\n|------|------|------|------|------|--------|-----|------|------|------|\n"
	rows := ""
	for i, d := range stockData {
		if i >= len(indicators) {
			break
		}
		row := fmt.Sprintf("| %s | %.2f | %.2f | %.2f | %.2f | %.0f | %.2f | %.2f | %.2f | %.2f |\n",
			d.Date.Format("2006-01-02"), d.Open, d.Close, d.High, d.Low, d.Volume,
			indicators[i].MA5, indicators[i].MA10, indicators[i].MA20, indicators[i].MA60)
		rows += row
		if i > 30 {
			break
		} // 只展示最近30天，防止prompt过长
	}
	return head + rows
}

// 只保留最近N个月的数据（支持动态起止）
func filterRecentDataToDate(stockData []StockData, indicators []TechnicalIndicator, endDate time.Time, months int) ([]StockData, []TechnicalIndicator) {
	if len(stockData) == 0 {
		return stockData, indicators
	}
	cutoff := endDate.AddDate(0, -months, 0)
	idx := 0
	for i, d := range stockData {
		if (d.Date.After(cutoff) || d.Date.Equal(cutoff)) && d.Date.Before(endDate.AddDate(0, 0, 1)) {
			idx = i
			break
		}
	}
	// 只保留截止endDate的半年数据
	var filteredData []StockData
	var filteredInd []TechnicalIndicator
	for i := idx; i < len(stockData); i++ {
		if stockData[i].Date.After(endDate) {
			break
		}
		filteredData = append(filteredData, stockData[i])
		filteredInd = append(filteredInd, indicators[i])
	}
	return filteredData, filteredInd
}

func AnalyzeOne(params AnalysisParams, genFunc func(string, string, string, string, string, bool, bool) (string, error)) AnalysisResult {
	prompt := params.Prompt
	if prompt == "" {
		prompt = BuildPrompt(params)
	}

	// 生成图表
	stockData, indicators, _ := FetchStockHistory(params.StockCodes[0], params.Start, params.End, params.APIKey)
	if len(stockData) > 0 {
		latest := stockData[len(stockData)-1].Date
		// fmt.Fprintf(os.Stderr, "[调试] 实际可用K线最新日期: %s\n", latest.Format("2006-01-02"))
		stockData, indicators = filterRecentDataToDate(stockData, indicators, latest, 12)
	}
	chartPaths, _ := GenerateCharts(params.StockCodes[0], stockData, indicators, "charts")

	// 新增：行情表格注入
	stockTable := FormatStockDataTable(stockData, indicators)
	prompt = stockTable + "\n" + prompt

	report, err := genFunc(params.StockCodes[0], prompt, params.APIKey, "https://api.deepseek.com/v1/chat/completions", params.Model, params.SearchMode, params.HybridSearch)
	if err != nil {
		return AnalysisResult{StockCode: params.StockCodes[0], Err: err}
	}

	// 在报告前插入图表引用
	var chartRefs string
	for _, p := range chartPaths {
		chartRefs += fmt.Sprintf("![图表](%s)\n", p)
	}
	report = chartRefs + "\n" + report

	// 自动修复：确保history目录存在
	os.MkdirAll("history", 0755)

	// 支持多格式导出
	exports := []string{"md"} // 默认md
	if len(params.Output) > 0 {
		exports = params.Output
	}
	var savedFile string
	var writeErr error
	for _, ext := range exports {
		var fname string
		fbase := fmt.Sprintf("%s-%s-%s", params.StockCodes[0], params.End, time.Now().Format("150405"))
		fpath := ""
		// 先处理图片
		reportHTML := replaceImagesWithAbsHTML(report)
		if ext == "md" {
			fname = fbase + ".md"
			fpath = filepath.Join("history", fname)
			err := ioutil.WriteFile(fpath, []byte(report), 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[错误] 写入Markdown文件失败: %s\n", err)
				writeErr = err
			} else {
				savedFile = fname
			}
		} else if ext == "html" {
			fname = fbase + ".html"
			fpath = filepath.Join("history", fname)
			html := "<meta charset=\"utf-8\">\n" + exportCSS + markdownToHTML(reportHTML)
			err := ioutil.WriteFile(fpath, []byte(html), 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[错误] 写入HTML文件失败: %s\n", err)
				writeErr = err
			} else {
				savedFile = fname
			}
		} else if ext == "pdf" {
			fname = fbase + ".pdf"
			fpath = filepath.Join("history", fname)
			htmlPath := fpath + ".tmp.html"
			htmlContent := "<meta charset=\"utf-8\">\n" + exportCSS + markdownToHTML(reportHTML)
			ioutil.WriteFile(htmlPath, []byte(htmlContent), 0644)
			err := htmlToPDF(htmlPath, fpath)
			os.Remove(htmlPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[错误] 生成PDF失败: %s\n", err)
				writeErr = err
			} else {
				fmt.Println("[调试] 已写入PDF文件：", fpath)
				savedFile = fname
			}
		}
	}
	if writeErr != nil {
		return AnalysisResult{StockCode: params.StockCodes[0], Report: report, SavedFile: savedFile, Err: writeErr}
	}
	return AnalysisResult{StockCode: params.StockCodes[0], Report: report, SavedFile: savedFile}
}

// printStepBoxStr 返回美观的框包裹字符串（不直接输出）
func getBoxWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 120 // fallback
	}
	bw := int(float64(w) * 0.8)
	if bw < 60 {
		bw = 60
	}
	if bw > 200 {
		bw = 200
	}
	return bw
}

func printStepBoxStr(title string, lines ...string) string {
	width := getBoxWidth()
	titleWidth := len([]rune(title))
	sideLen := (width - 2 - titleWidth) / 2
	top := "┌" + strings.Repeat("─", sideLen) + title + strings.Repeat("─", width-2-titleWidth-sideLen) + "┐\n"
	body := ""
	for _, l := range lines {
		maxContent := width - 2
		lWidth := len([]rune(l))
		if lWidth > maxContent {
			l = l[:maxContent-1] + "…"
			lWidth = len([]rune(l))
		}
		pad := maxContent - lWidth
		if pad < 0 {
			pad = 0
		}
		body += fmt.Sprintf("│%s%s│\n", l, strings.Repeat(" ", pad))
	}
	bottom := "└" + strings.Repeat("─", width-2) + "┘\n"
	return top + body + bottom
}

func htmlToPDF(htmlPath, pdfPath string) error {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	var pdfBuf []byte
	absPath, _ := filepath.Abs(htmlPath)
	fileURL := "file://" + absPath
	err := chromedp.Run(ctx,
		chromedp.Navigate(fileURL),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfBuf, _, err = page.PrintToPDF().WithPrintBackground(true).Do(ctx)
			return err
		}),
	)
	if err != nil {
		return err
	}
	return os.WriteFile(pdfPath, pdfBuf, 0644)
}

const exportCSS = `
<style>
body { font-family: 'SF Pro', 'Arial', 'Microsoft YaHei', sans-serif; font-size: 15px; margin: 24px; }
h1,h2,h3,h4 { font-weight: bold; margin: 1.2em 0 0.6em 0; }
table { border-collapse: collapse; width: 100%; margin: 1em 0; }
th, td { border: 1px solid #888; padding: 8px 12px; text-align: left; }
th { background: #f0f0f0; }
pre { background: #222; color: #fff; border-radius: 6px; border: 1px solid #444; padding: 14px; font-size: 15px; overflow-x: auto; margin: 1em 0; font-family: monospace; white-space: pre-wrap; }
strong { font-weight: bold; }
img { display: block; margin: 18px auto; max-width: 95%; border-radius: 4px; box-shadow: 0 2px 8px #0001; }
ul,ol { margin: 1em 0 1em 2em; }
code { background: #f5f5f5; color: #c7254e; padding: 2px 4px; border-radius: 4px; }
</style>
`

// 修改 GenerateAIReportWithConfigAndSearch 实现，支持 hybridSearch
func GenerateAIReportWithConfigAndSearch(stock, prompt, apiKey, apiURL, model string, searchMode bool, hybridSearch bool) (string, error) {
	// 构造请求体
	body := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "你是一个智能股票分析助手。"},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
		"max_tokens":  2000,
	}
	if hybridSearch {
		body["search"] = true // 混合模式，自动融合
	} else if searchMode {
		body["search"] = true // 兼容原有联网搜索
	}
	data, _ := json.Marshal(body)
	client := &http.Client{}
	req, _ := http.NewRequest("POST", apiURL, strings.NewReader(string(data)))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respData, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("DeepSeek API 错误: %s", string(respData))
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	err = json.Unmarshal(respData, &result)
	if err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("DeepSeek API 无返回内容")
	}
	return result.Choices[0].Message.Content, nil
}

// 字符串宽度填充，兼容中英文
func pad(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w < width {
		return s + strings.Repeat(" ", width-w)
	}
	if w > width {
		rs := []rune(s)
		cut := 0
		for i := range rs {
			if runewidth.StringWidth(string(rs[:i+1])) > width {
				break
			}
			cut = i + 1
		}
		return string(rs[:cut])
	}
	return s
}
