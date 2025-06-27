package main

import (
	"Quantix/analysis"
	"Quantix/data"
	"Quantix/config"
	"Quantix/api"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "web" {
		// 启动Web API服务器
		cfg, err := config.LoadConfig("config.yaml")
		if err != nil {
			panic(err)
		}
		server := api.NewServer(cfg)
		if err := server.Start(); err != nil {
			panic(err)
		}
		return
	}

	if len(os.Args) < 2 {
		fmt.Println("用法:")
		fmt.Println("  单股票分析: Quantix <股票代码> [开始日期] [结束日期]")
		fmt.Println("  多股票对比: Quantix <股票代码1,股票代码2,...> [开始日期] [结束日期] [--factors=因子1,因子2,...] [--weights=w1,w2,...] [--csv=文件1,文件2,...]")
		fmt.Println("示例:")
		fmt.Println("  Quantix AAPL 2023-01-01 2023-12-31")
		fmt.Println("  Quantix 000001.SZ,600519.SH 2024-01-01 2024-04-01 --factors=Momentum,RSI,MACD --weights=0.3,0.4,0.3 --csv=AAPL.csv,MSFT.csv")
		return
	}

	codes := strings.Split(os.Args[1], ",")
	start := "2023-01-01"
	end := "2023-12-31"
	factorsArg := ""
	weightsArg := ""
	csvArg := ""
	if len(os.Args) >= 4 {
		start = os.Args[2]
		end = os.Args[3]
	}
	// 解析额外参数
	for _, arg := range os.Args[4:] {
		if strings.HasPrefix(arg, "--factors=") {
			factorsArg = strings.TrimPrefix(arg, "--factors=")
		}
		if strings.HasPrefix(arg, "--weights=") {
			weightsArg = strings.TrimPrefix(arg, "--weights=")
		}
		if strings.HasPrefix(arg, "--csv=") {
			csvArg = strings.TrimPrefix(arg, "--csv=")
		}
	}

	if len(codes) == 1 {
		// 单股票分析
		analyzeSingleStockWithCSV(codes[0], start, end, csvArg)
	} else {
		// 多股票对比分析
		analyzeMultipleStocksWithFactorsAndCSV(codes, start, end, factorsArg, weightsArg, csvArg)
	}
}

// 单股票分析，支持本地CSV
func analyzeSingleStockWithCSV(code, start, end, csvArg string) {
	var klines []data.Kline
	var err error
	if csvArg != "" {
		// 只取第一个csv文件
		csvFiles := strings.Split(csvArg, ",")
		klines, err = data.FetchCSVKlines(csvFiles[0])
		if err != nil {
			fmt.Println("本地CSV读取失败:", err)
			return
		}
		fmt.Printf("[本地CSV] 读取 %s，共 %d 条K线数据\n", csvFiles[0], len(klines))
	} else if len(code) > 3 && (code[len(code)-3:] == ".SZ" || code[len(code)-3:] == ".SH") {
		// A股
		startDate, _ := time.Parse("2006-01-02", start)
		endDate, _ := time.Parse("2006-01-02", end)
		startStr := startDate.Format("2006-01-02")
		endStr := endDate.Format("2006-01-02")
		fmt.Printf("[A股] 获取 %s 从 %s 到 %s 的行情数据...\n", code, startStr, endStr)
		klines, err = data.FetchTencentKlines(code, startStr, endStr)
	} else {
		// 美股/港股
		fmt.Printf("[美股/港股] 获取 %s 从 %s 到 %s 的行情数据...\n", code, start, end)
		klines, err = data.FetchYahooKlines(code, start, end)
	}

	if err != nil {
		fmt.Println("数据获取失败:", err)
		return
	}

	if len(klines) < 30 {
		fmt.Println("数据不足30天，无法进行完整分析")
		return
	}

	fmt.Printf("成功获取 %d 条K线数据\n", len(klines))

	// 技术指标分析
	factors := analysis.CalcFactors(klines)
	if len(factors) > 0 {
		// 设置股票代码
		for i := range factors {
			factors[i].Code = code
		}
		analysis.PrintAnalysis(factors)
	}

	// 风险管理分析
	fmt.Println("\n" + strings.Repeat("=", 50))
	riskMetrics := analysis.CalculateRiskMetrics(klines, 0.03) // 假设无风险利率3%
	analysis.PrintRiskMetrics(riskMetrics)

	// 机器学习预测
	fmt.Println("\n" + strings.Repeat("=", 50))
	if len(factors) > 0 {
		predictor := analysis.NewMLPredictor(klines, factors)
		mlPredictions := predictor.PredictAll()
		analysis.PrintMLPredictions(mlPredictions)
	}

	// 传统趋势预测
	fmt.Println("\n" + strings.Repeat("=", 50))
	if len(klines) >= 30 {
		prediction := analysis.PredictTrend(klines)
		analysis.PrintPrediction(prediction, code)

		// 多策略回测
		fmt.Println()
		analysis.RunMultiStrategyBacktest(klines)

		// 生成图表
		fmt.Println("\n=== 生成图表 ===")

		// 生成K线图
		if err := analysis.GenerateKlineChart(klines, code, start, end); err != nil {
			fmt.Printf("生成K线图失败: %v\n", err)
		}

		// 执行单次回测用于图表生成
		config := analysis.BacktestConfig{
			InitialCapital: 100000,
			PositionSize:   0.7,
			StopLoss:       0.08,
			TakeProfit:     0.15,
			MinHoldDays:    3,
			MaxHoldDays:    15,
		}
		backtestResult := analysis.RunBacktest(klines, config)

		// 生成回测图表
		if err := analysis.GenerateBacktestChart(klines, backtestResult, code, start, end); err != nil {
			fmt.Printf("生成回测图表失败: %v\n", err)
		}

		// 生成综合分析图表
		if len(factors) > 0 {
			latestFactors := factors[len(factors)-1]
			if err := analysis.GenerateAnalysisChart(klines, latestFactors, prediction, backtestResult, code, start, end); err != nil {
				fmt.Printf("生成综合分析图表失败: %v\n", err)
			}
		}

		fmt.Println("图表生成完成！请在charts目录中查看HTML文件。")
	} else {
		fmt.Println("\n数据不足30天，无法进行趋势预测和回测")
	}
}

// 多股票对比分析，支持本地CSV
func analyzeMultipleStocksWithFactorsAndCSV(codes []string, start, end, factorsArg, weightsArg, csvArg string) {
	fmt.Printf("开始多股票对比分析，共 %d 只股票\n", len(codes))

	stockData := make(map[string][]data.Kline)
	csvFiles := map[string]string{}
	if csvArg != "" {
		csvArr := strings.Split(csvArg, ",")
		for i, csvFile := range csvArr {
			if i < len(codes) {
				csvFiles[codes[i]] = csvFile
			}
		}
	}

	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}

		var klines []data.Kline
		var err error
		if csvFile, ok := csvFiles[code]; ok && csvFile != "" {
			klines, err = data.FetchCSVKlines(csvFile)
			if err != nil {
				fmt.Printf("本地CSV读取失败(%s): %v\n", csvFile, err)
				continue
			}
			fmt.Printf("[本地CSV] 读取 %s，共 %d 条K线数据\n", csvFile, len(klines))
		} else if len(code) > 3 && (code[len(code)-3:] == ".SZ" || code[len(code)-3:] == ".SH") {
			// A股
			startDate, _ := time.Parse("2006-01-02", start)
			endDate, _ := time.Parse("2006-01-02", end)
			startStr := startDate.Format("2006-01-02")
			endStr := endDate.Format("2006-01-02")
			fmt.Printf("[A股] 获取 %s 从 %s 到 %s 的行情数据...\n", code, startStr, endStr)
			klines, err = data.FetchTencentKlines(code, startStr, endStr)
		} else {
			// 美股/港股
			fmt.Printf("[美股/港股] 获取 %s 从 %s 到 %s 的行情数据...\n", code, start, end)
			klines, err = data.FetchYahooKlines(code, start, end)
		}

		if err != nil {
			fmt.Printf("获取 %s 数据失败: %v\n", code, err)
			continue
		}

		stockData[code] = klines
		fmt.Printf("成功获取 %s 数据，共 %d 条记录\n", code, len(klines))
	}

	if len(stockData) == 0 {
		fmt.Println("没有成功获取任何股票数据")
		return
	}

	// 解析因子和权重
	var factors []analysis.FactorWeight
	if factorsArg != "" {
		fNames := strings.Split(factorsArg, ",")
		var weights []float64
		if weightsArg != "" {
			wStrs := strings.Split(weightsArg, ",")
			for _, w := range wStrs {
				f, err := strconv.ParseFloat(w, 64)
				if err == nil {
					weights = append(weights, f)
				}
			}
		}
		for i, name := range fNames {
			fw := analysis.FactorWeight{Name: name, Weight: 0.0}
			if i < len(weights) {
				fw.Weight = weights[i]
			} else {
				fw.Weight = 1.0 / float64(len(fNames)) // 未指定权重时均分
			}
			factors = append(factors, fw)
		}
		// 权重归一化
		sum := 0.0
		for _, fw := range factors {
			sum += fw.Weight
		}
		if sum > 0 {
			for i := range factors {
				factors[i].Weight /= sum
			}
		}
	} else {
		factors = analysis.DefaultFactors
	}

	// 多因子打分与排名
	comparison := analysis.ScoreStocksByFactors(stockData, factors)
	analysis.PrintComparison(comparison)

	// 风险对比分析
	fmt.Println("\n=== 风险指标对比 ===")
	for code, klines := range stockData {
		if len(klines) >= 30 {
			fmt.Printf("\n--- %s 风险分析 ---\n", code)
			riskMetrics := analysis.CalculateRiskMetrics(klines, 0.03)
			analysis.PrintRiskMetrics(riskMetrics)
		}
	}
}
