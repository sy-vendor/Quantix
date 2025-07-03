package analysis

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// GenerateCharts 自动生成K线、均线、成交量图，返回PNG图片路径列表
func GenerateCharts(stockCode string, stockData []StockData, indicators []TechnicalIndicator, outDir string) ([]string, error) {
	if len(stockData) == 0 {
		return nil, nil
	}
	os.MkdirAll(outDir, 0755)
	var paths []string

	// 1. K线图
	kline := charts.NewKLine()
	var kDates []string
	var kItems []opts.KlineData
	for _, d := range stockData {
		kDates = append(kDates, d.Date.Format("2006-01-02"))
		kItems = append(kItems, opts.KlineData{
			Value: [4]float64{d.Open, d.Close, d.Low, d.High},
		})
	}
	kline.SetGlobalOptions()
	kline.SetXAxis(kDates).AddSeries("K线", kItems)
	klinePath := filepath.Join(outDir, stockCode+"-kline.html")
	f1, _ := os.Create(klinePath)
	_ = kline.Render(f1)
	klinePNG := filepath.Join(outDir, stockCode+"-kline.png")
	_ = html2png(klinePath, klinePNG)
	paths = append(paths, klinePNG)

	// 2. 均线图
	ma := charts.NewLine()
	var ma5, ma10, ma20, ma60 []opts.LineData
	for _, ind := range indicators {
		ma5 = append(ma5, opts.LineData{Value: ind.MA5})
		ma10 = append(ma10, opts.LineData{Value: ind.MA10})
		ma20 = append(ma20, opts.LineData{Value: ind.MA20})
		ma60 = append(ma60, opts.LineData{Value: ind.MA60})
	}
	ma.SetGlobalOptions()
	ma.SetXAxis(kDates).
		AddSeries("MA5", ma5).
		AddSeries("MA10", ma10).
		AddSeries("MA20", ma20).
		AddSeries("MA60", ma60)
	maPath := filepath.Join(outDir, stockCode+"-ma.html")
	f2, _ := os.Create(maPath)
	_ = ma.Render(f2)
	maPNG := filepath.Join(outDir, stockCode+"-ma.png")
	_ = html2png(maPath, maPNG)
	paths = append(paths, maPNG)

	// 3. 成交量图
	vol := charts.NewBar()
	var vols []opts.BarData
	for _, d := range stockData {
		vols = append(vols, opts.BarData{Value: d.Volume})
	}
	vol.SetGlobalOptions()
	vol.SetXAxis(kDates).AddSeries("成交量", vols)
	volPath := filepath.Join(outDir, stockCode+"-vol.html")
	f3, _ := os.Create(volPath)
	_ = vol.Render(f3)
	volPNG := filepath.Join(outDir, stockCode+"-vol.png")
	_ = html2png(volPath, volPNG)
	paths = append(paths, volPNG)

	return paths, nil
}

// html2png 用 chromedp 将 HTML 渲染为 PNG
func html2png(htmlPath, pngPath string) error {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var buf []byte
	absPath, _ := filepath.Abs(htmlPath)
	fileURL := "file://" + absPath
	// 让 go-echarts 图表自适应宽度
	err := chromedp.Run(ctx,
		chromedp.Navigate(fileURL),
		chromedp.Sleep(500*time.Millisecond), // 等待渲染
		chromedp.FullScreenshot(&buf, 100),
	)
	if err != nil {
		return err
	}
	return os.WriteFile(pngPath, buf, 0644)
}
