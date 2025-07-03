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

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/russross/blackfriday/v2"
)

// 类型定义补充

// AnalysisParams 用于传递分析参数
// 可根据main.go和ai.go的实际用法调整字段
// StockData、TechnicalIndicator用于chart.go

type AnalysisParams struct {
	APIKey     string
	Model      string
	StockCodes []string
	Start      string
	End        string
	SearchMode bool
	Periods    []string
	Dims       []string
	Output     []string
	Confidence bool
	Risk       string
	Scope      []string
	Lang       string
	Prompt     string // 可选，手动传递prompt
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
	for i := range closes {
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
	langMap := map[string]string{"zh": "中文", "en": "英文"}
	lang := langMap[params.Lang]
	if lang == "" {
		lang = params.Lang
	}
	prompt := fmt.Sprintf(
		"请用%s分析股票【%s】在区间【%s ~ %s】的走势。\n",
		lang,
		params.StockCodes[0],
		params.Start, params.End,
	)
	if len(params.Periods) > 0 {
		prompt += fmt.Sprintf("分析周期包括：%s。\n", strings.Join(params.Periods, "、"))
	}
	if len(params.Dims) > 0 {
		prompt += fmt.Sprintf("分析维度包括：%s。\n", strings.Join(params.Dims, "、"))
	}
	prompt += "请结合上方K线图、均线图、成交量图，对当前股票的走势、支撑阻力、均线形态、量价关系等进行详细分析，给出趋势判断、操作建议和风险提示。分析内容需包括：\n"
	prompt += "1. K线形态与趋势解读\n"
	prompt += "2. 均线系统排列与金叉死叉分析\n"
	prompt += "3. 成交量变化与量价关系\n"
	prompt += "4. 支撑位与阻力位判断\n"
	prompt += "5. 操作建议（如持有/加仓/减仓/观望等）\n"
	prompt += "6. 风险提示与注意事项\n"
	prompt += "请用专业、客观、结构化的语言输出，适合投资者决策参考。"
	return prompt
}

func markdownToHTML(md string) string {
	html := blackfriday.Run([]byte(md))
	return string(html)
}

func AnalyzeOne(params AnalysisParams, genFunc func(string, string, string, string, string, bool) (string, error)) AnalysisResult {
	prompt := params.Prompt
	if prompt == "" {
		prompt = BuildPrompt(params)
	}

	// 生成图表
	stockData, indicators, _ := FetchStockHistory(params.StockCodes[0], params.Start, params.End, params.APIKey)
	chartPaths, _ := GenerateCharts(params.StockCodes[0], stockData, indicators, "charts")

	report, err := genFunc(params.StockCodes[0], prompt, params.APIKey, "https://api.deepseek.com/v1/chat/completions", params.Model, params.SearchMode)
	if err != nil {
		return AnalysisResult{StockCode: params.StockCodes[0], Err: err}
	}

	// 在报告前插入图表引用
	var chartRefs string
	for _, p := range chartPaths {
		chartRefs += fmt.Sprintf("![图表](%s)\n", p)
	}
	report = chartRefs + report

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
		if ext == "md" {
			fname = fbase + ".md"
			fpath = filepath.Join("history", fname)
			err := ioutil.WriteFile(fpath, []byte(report), 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[错误] 写入Markdown文件失败: %s\n", err)
				writeErr = err
			} else {
				fmt.Println("[调试] 已写入Markdown文件：", fpath)
				savedFile = fname
			}
		} else if ext == "html" {
			fname = fbase + ".html"
			fpath = filepath.Join("history", fname)
			html := "<meta charset=\"utf-8\">\n" + markdownToHTML(report)
			err := ioutil.WriteFile(fpath, []byte(html), 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[错误] 写入HTML文件失败: %s\n", err)
				writeErr = err
			} else {
				fmt.Println("[调试] 已写入HTML文件：", fpath)
				savedFile = fname
			}
		} else if ext == "pdf" {
			fname = fbase + ".pdf"
			fpath = filepath.Join("history", fname)
			html := "<meta charset=\"utf-8\">\n" + markdownToHTML(report)
			pdfg, _ := wkhtmltopdf.NewPDFGenerator()
			pdfg.AddPage(wkhtmltopdf.NewPageReader(strings.NewReader(html)))
			pdfg.Dpi.Set(300)
			pdfg.Orientation.Set(wkhtmltopdf.OrientationPortrait)
			pdfg.PageSize.Set(wkhtmltopdf.PageSizeA4)
			pdfg.NoCollate.Set(false)
			pdfg.Grayscale.Set(false)
			pdfg.MarginLeft.Set(10)
			pdfg.MarginRight.Set(10)
			pdfg.MarginTop.Set(10)
			pdfg.MarginBottom.Set(10)
			err := pdfg.Create()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[错误] 生成PDF失败: %s\n", err)
				writeErr = err
				continue
			}
			err = pdfg.WriteFile(fpath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[错误] 写入PDF文件失败: %s\n", err)
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
