package analysis

import (
	"Quantix/data"
	"fmt"
	"sort"
)

type StockScore struct {
	Code      string
	Name      string
	Factors   Factors
	Score     float64
	Rank      int
	Recommend string
}

type StockComparison struct {
	Stocks []StockScore
}

// 计算综合评分
func calculateScore(f Factors) float64 {
	score := 0.0

	// RSI评分 (0-100)
	if f.RSI > 30 && f.RSI < 70 {
		score += 25 // RSI在正常区间
	} else if f.RSI >= 70 {
		score += 10 // 超买，分数较低
	} else {
		score += 15 // 超卖，中等分数
	}

	// MACD评分
	if f.MACD > f.MACDSignal {
		score += 25 // MACD金叉
	} else {
		score += 10 // MACD死叉
	}

	// 均线评分
	if f.MA5 > f.MA10 && f.MA10 > f.MA20 {
		score += 25 // 多头排列
	} else if f.MA5 < f.MA10 && f.MA10 < f.MA20 {
		score += 5 // 空头排列
	} else {
		score += 15 // 均线混乱
	}

	// 换手率评分
	if f.Turnover > 0.8 && f.Turnover < 1.5 {
		score += 15 // 正常换手
	} else if f.Turnover >= 1.5 {
		score += 10 // 高换手，注意风险
	} else {
		score += 5 // 低换手
	}

	// 动量评分
	if f.Momentum > 0 {
		score += 10 // 上涨
	} else {
		score += 0 // 下跌
	}

	return score
}

// 生成投资建议
func generateRecommendation(score float64, f Factors) string {
	if score >= 80 {
		return "强烈推荐"
	} else if score >= 60 {
		return "推荐"
	} else if score >= 40 {
		return "观望"
	} else {
		return "谨慎"
	}
}

// 分析多只股票
func AnalyzeMultipleStocks(stockData map[string][]data.Kline) StockComparison {
	var comparison StockComparison

	for code, klines := range stockData {
		if len(klines) == 0 {
			continue
		}

		factors := CalcFactors(klines)
		if len(factors) == 0 {
			continue
		}

		// 使用最新的技术指标
		latestFactors := factors[len(factors)-1]
		score := calculateScore(latestFactors)
		recommend := generateRecommendation(score, latestFactors)

		stockScore := StockScore{
			Code:      code,
			Name:      getStockName(code),
			Factors:   latestFactors,
			Score:     score,
			Recommend: recommend,
		}

		comparison.Stocks = append(comparison.Stocks, stockScore)
	}

	// 按评分排序
	sort.Slice(comparison.Stocks, func(i, j int) bool {
		return comparison.Stocks[i].Score > comparison.Stocks[j].Score
	})

	// 设置排名
	for i := range comparison.Stocks {
		comparison.Stocks[i].Rank = i + 1
	}

	return comparison
}

// 获取股票名称（简化版，实际可以从数据库或API获取）
func getStockName(code string) string {
	nameMap := map[string]string{
		"000001.SZ": "平安银行",
		"600519.SH": "贵州茅台",
		"AAPL":      "苹果公司",
		"TSLA":      "特斯拉",
		"MSFT":      "微软",
		"GOOGL":     "谷歌",
	}
	if name, exists := nameMap[code]; exists {
		return name
	}
	return code
}

// 打印对比分析结果
func PrintComparison(comparison StockComparison) {
	fmt.Println("=== 多股票对比分析 ===")
	fmt.Printf("分析股票数量: %d\n\n", len(comparison.Stocks))

	fmt.Println("--- 综合排名 ---")
	for _, stock := range comparison.Stocks {
		fmt.Printf("%d. %s(%s) - 评分: %.1f - %s\n",
			stock.Rank, stock.Name, stock.Code, stock.Score, stock.Recommend)
	}

	fmt.Println("\n--- 详细对比 ---")
	fmt.Printf("%-12s %-8s %-8s %-8s %-8s %-8s %-8s %-8s\n",
		"股票代码", "5日均线", "10日均线", "RSI", "MACD", "换手率", "评分", "建议")
	fmt.Println("--------------------------------------------------------------------------------")

	for _, stock := range comparison.Stocks {
		fmt.Printf("%-12s %-8.2f %-8.2f %-8.2f %-8.4f %-8.2f %-8.1f %-8s\n",
			stock.Code,
			stock.Factors.MA5,
			stock.Factors.MA10,
			stock.Factors.RSI,
			stock.Factors.MACD,
			stock.Factors.Turnover,
			stock.Score,
			stock.Recommend)
	}

	// 投资组合建议
	fmt.Println("\n--- 投资组合建议 ---")
	recommended := 0
	for _, stock := range comparison.Stocks {
		if stock.Score >= 60 {
			recommended++
		}
	}

	if recommended > 0 {
		fmt.Printf("推荐股票数量: %d\n", recommended)
		fmt.Println("推荐股票列表:")
		for _, stock := range comparison.Stocks {
			if stock.Score >= 60 {
				fmt.Printf("  - %s(%s): %.1f分\n", stock.Name, stock.Code, stock.Score)
			}
		}
	} else {
		fmt.Println("当前市场环境下，建议谨慎投资，可考虑观望或降低仓位。")
	}
}

// 多因子打分参数结构
type FactorWeight struct {
	Name   string
	Weight float64
}

// 支持的所有因子及默认权重
var DefaultFactors = []FactorWeight{
	{"Momentum", 0.20},
	{"Volatility", 0.10},
	{"MA5", 0.10},
	{"RSI", 0.20},
	{"MACD", 0.20},
	{"Turnover", 0.10},
	{"OBV", 0.10},
}

// 多因子归一化打分
func ScoreStocksByFactors(stockData map[string][]data.Kline, factors []FactorWeight) StockComparison {
	// 1. 计算所有股票的最新因子
	type factorRaw struct {
		Code   string
		Values map[string]float64
		Origin Factors
	}

	var allRaw []factorRaw
	for code, klines := range stockData {
		if len(klines) == 0 {
			continue
		}
		factorsArr := CalcFactors(klines)
		if len(factorsArr) == 0 {
			continue
		}
		latest := factorsArr[len(factorsArr)-1]
		values := map[string]float64{
			"Momentum":   latest.Momentum,
			"Volatility": latest.Volatility,
			"MA5":        latest.MA5,
			"RSI":        latest.RSI,
			"MACD":       latest.MACD,
			"Turnover":   latest.Turnover,
			"OBV":        latest.OBV,
		}
		allRaw = append(allRaw, factorRaw{Code: code, Values: values, Origin: latest})
	}

	if len(allRaw) == 0 {
		return StockComparison{}
	}

	// 2. 对每个因子归一化（min-max）
	normed := make([]map[string]float64, len(allRaw))
	for i := range normed {
		normed[i] = make(map[string]float64)
	}
	for _, fw := range factors {
		minV, maxV := 1e12, -1e12
		for _, row := range allRaw {
			v := row.Values[fw.Name]
			if v < minV {
				minV = v
			}
			if v > maxV {
				maxV = v
			}
		}
		for i, row := range allRaw {
			v := row.Values[fw.Name]
			if maxV > minV {
				normed[i][fw.Name] = (v - minV) / (maxV - minV)
			} else {
				normed[i][fw.Name] = 0.5 // 全部一样时给0.5
			}
		}
	}

	// 3. 计算加权总分
	var comparison StockComparison
	for i, row := range allRaw {
		score := 0.0
		for _, fw := range factors {
			score += normed[i][fw.Name] * fw.Weight
		}
		stockScore := StockScore{
			Code:    row.Code,
			Name:    getStockName(row.Code),
			Factors: row.Origin,
			Score:   score * 100, // 归一化后乘100
		}
		comparison.Stocks = append(comparison.Stocks, stockScore)
	}

	// 4. 排序和排名
	sort.Slice(comparison.Stocks, func(i, j int) bool {
		return comparison.Stocks[i].Score > comparison.Stocks[j].Score
	})
	for i := range comparison.Stocks {
		comparison.Stocks[i].Rank = i + 1
	}

	return comparison
}

// CompareStocks 统一对外接口
func CompareStocks(codes []string, start, end string, factors []string, weights []float64) StockComparison {
	// 1. 获取每只股票的K线数据
	stockData := make(map[string][]data.Kline)
	for _, code := range codes {
		klines, err := data.FetchYahooKlines(code, start, end)
		if err == nil && len(klines) > 0 {
			stockData[code] = klines
		}
	}
	// 2. 默认用 AnalyzeMultipleStocks
	return AnalyzeMultipleStocks(stockData)
}
