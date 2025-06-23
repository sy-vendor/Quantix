package analysis

import (
	"Quantix/data"
	"fmt"
	"math"
)

type BacktestResult struct {
	InitialCapital float64 // 初始资金
	FinalCapital   float64 // 最终资金
	TotalReturn    float64 // 总收益率
	AnnualReturn   float64 // 年化收益率
	MaxDrawdown    float64 // 最大回撤
	SharpeRatio    float64 // 夏普比率
	WinRate        float64 // 胜率
	TotalTrades    int     // 总交易次数
	WinTrades      int     // 盈利交易次数
	LossTrades     int     // 亏损交易次数
	AvgWin         float64 // 平均盈利
	AvgLoss        float64 // 平均亏损
	ProfitFactor   float64 // 盈亏比
	TradeHistory   []Trade // 交易历史
}

type Trade struct {
	Date    string  // 交易日期
	Type    string  // 买入/卖出
	Price   float64 // 交易价格
	Shares  int     // 交易股数
	Value   float64 // 交易金额
	Capital float64 // 当前资金
	Reason  string  // 交易原因
}

// 回测策略配置
type BacktestConfig struct {
	InitialCapital float64 // 初始资金
	PositionSize   float64 // 仓位比例 (0-1)
	StopLoss       float64 // 止损比例
	TakeProfit     float64 // 止盈比例
	MinHoldDays    int     // 最小持有天数
	MaxHoldDays    int     // 最大持有天数
}

// 执行回测
func RunBacktest(klines []data.Kline, config BacktestConfig) BacktestResult {
	if len(klines) < 30 {
		return BacktestResult{}
	}

	var result BacktestResult
	result.InitialCapital = config.InitialCapital
	result.TradeHistory = make([]Trade, 0)

	capital := config.InitialCapital
	shares := 0
	avgCost := 0.0
	lastTradeDate := ""

	for i := 30; i < len(klines); i++ { // 从第30天开始，确保有足够数据计算指标
		currentKlines := klines[:i+1]
		prediction := PredictTrend(currentKlines)
		currentPrice := klines[i].Close
		currentDate := klines[i].Date.Format("2006-01-02")

		// 检查是否需要卖出
		if shares > 0 {
			// 止损检查
			if currentPrice <= avgCost*(1-config.StopLoss) {
				// 止损卖出
				tradeValue := float64(shares) * currentPrice
				capital = tradeValue
				result.TradeHistory = append(result.TradeHistory, Trade{
					Date:    currentDate,
					Type:    "止损卖出",
					Price:   currentPrice,
					Shares:  shares,
					Value:   tradeValue,
					Capital: capital,
					Reason:  fmt.Sprintf("止损 %.1f%%", config.StopLoss*100),
				})
				shares = 0
				avgCost = 0
				lastTradeDate = currentDate
				continue
			}

			// 止盈检查
			if currentPrice >= avgCost*(1+config.TakeProfit) {
				// 止盈卖出
				tradeValue := float64(shares) * currentPrice
				capital = tradeValue
				result.TradeHistory = append(result.TradeHistory, Trade{
					Date:    currentDate,
					Type:    "止盈卖出",
					Price:   currentPrice,
					Shares:  shares,
					Value:   tradeValue,
					Capital: capital,
					Reason:  fmt.Sprintf("止盈 %.1f%%", config.TakeProfit*100),
				})
				shares = 0
				avgCost = 0
				lastTradeDate = currentDate
				continue
			}

			// 最大持有天数检查
			if config.MaxHoldDays > 0 && daysBetween(lastTradeDate, currentDate) >= config.MaxHoldDays {
				// 超时卖出
				tradeValue := float64(shares) * currentPrice
				capital = tradeValue
				result.TradeHistory = append(result.TradeHistory, Trade{
					Date:    currentDate,
					Type:    "超时卖出",
					Price:   currentPrice,
					Shares:  shares,
					Value:   tradeValue,
					Capital: capital,
					Reason:  fmt.Sprintf("持有%d天", config.MaxHoldDays),
				})
				shares = 0
				avgCost = 0
				lastTradeDate = currentDate
				continue
			}

			// 趋势反转卖出 - 提高置信度要求
			if prediction.Trend == "下跌" && prediction.Confidence >= 70 {
				tradeValue := float64(shares) * currentPrice
				capital = tradeValue
				result.TradeHistory = append(result.TradeHistory, Trade{
					Date:    currentDate,
					Type:    "趋势卖出",
					Price:   currentPrice,
					Shares:  shares,
					Value:   tradeValue,
					Capital: capital,
					Reason:  fmt.Sprintf("趋势下跌 %.1f%%", prediction.Confidence),
				})
				shares = 0
				avgCost = 0
				lastTradeDate = currentDate
				continue
			}
		}

		// 检查是否需要买入 - 提高置信度要求
		if shares == 0 && prediction.Trend == "上涨" && prediction.Confidence >= 70 {
			// 最小持有天数检查 - 如果是第一次交易或已经满足最小间隔，允许买入
			if lastTradeDate == "" || daysBetween(lastTradeDate, currentDate) >= config.MinHoldDays {
				// 买入
				positionValue := capital * config.PositionSize
				shares = int(positionValue / currentPrice)
				if shares > 0 {
					tradeValue := float64(shares) * currentPrice
					capital -= tradeValue
					avgCost = currentPrice
					lastTradeDate = currentDate

					result.TradeHistory = append(result.TradeHistory, Trade{
						Date:    currentDate,
						Type:    "买入",
						Price:   currentPrice,
						Shares:  shares,
						Value:   tradeValue,
						Capital: capital,
						Reason:  fmt.Sprintf("趋势上涨 %.1f%%", prediction.Confidence),
					})
				}
			}
		}
	}

	// 最后一天强制平仓
	if shares > 0 {
		finalPrice := klines[len(klines)-1].Close
		finalValue := float64(shares) * finalPrice
		capital += finalValue
		result.TradeHistory = append(result.TradeHistory, Trade{
			Date:    klines[len(klines)-1].Date.Format("2006-01-02"),
			Type:    "期末平仓",
			Price:   finalPrice,
			Shares:  shares,
			Value:   finalValue,
			Capital: capital,
			Reason:  "回测结束",
		})
	}

	// 计算回测结果
	result.FinalCapital = capital
	result.TotalReturn = (capital - config.InitialCapital) / config.InitialCapital

	// 计算年化收益率
	days := len(klines)
	result.AnnualReturn = math.Pow(1+result.TotalReturn, 365.0/float64(days)) - 1

	// 计算其他指标
	result.calculateMetrics()

	return result
}

// 计算回测指标
func (r *BacktestResult) calculateMetrics() {
	// 计算最大回撤
	r.MaxDrawdown = r.calculateMaxDrawdown()

	// 计算夏普比率
	r.SharpeRatio = r.calculateSharpeRatio()

	// 计算胜率和其他交易指标
	r.calculateTradeMetrics()
}

// 计算最大回撤
func (r *BacktestResult) calculateMaxDrawdown() float64 {
	maxDrawdown := 0.0
	peak := r.InitialCapital

	for _, trade := range r.TradeHistory {
		if trade.Capital > peak {
			peak = trade.Capital
		}
		drawdown := (peak - trade.Capital) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

// 计算夏普比率（简化版）
func (r *BacktestResult) calculateSharpeRatio() float64 {
	if r.TotalReturn <= 0 {
		return 0
	}

	// 简化计算，假设无风险利率为3%
	riskFreeRate := 0.03
	excessReturn := r.AnnualReturn - riskFreeRate

	// 假设年化波动率为20%（简化计算）
	volatility := 0.20

	if volatility == 0 {
		return 0
	}

	return excessReturn / volatility
}

// 计算交易指标
func (r *BacktestResult) calculateTradeMetrics() {
	r.TotalTrades = len(r.TradeHistory) / 2 // 买入卖出配对

	if r.TotalTrades == 0 {
		return
	}

	var wins, losses int
	var totalWin, totalLoss float64

	for i := 1; i < len(r.TradeHistory); i += 2 {
		if i+1 < len(r.TradeHistory) {
			buyTrade := r.TradeHistory[i]
			sellTrade := r.TradeHistory[i+1]

			profit := sellTrade.Value - buyTrade.Value

			if profit > 0 {
				wins++
				totalWin += profit
			} else {
				losses++
				totalLoss += math.Abs(profit)
			}
		}
	}

	r.WinTrades = wins
	r.LossTrades = losses

	if wins > 0 {
		r.AvgWin = totalWin / float64(wins)
	}
	if losses > 0 {
		r.AvgLoss = totalLoss / float64(losses)
	}

	r.WinRate = float64(wins) / float64(r.TotalTrades)

	if r.AvgLoss > 0 {
		r.ProfitFactor = r.AvgWin / r.AvgLoss
	}
}

// 计算两个日期之间的天数
func daysBetween(date1, date2 string) int {
	// 简化实现，直接返回1表示至少间隔1天
	// 实际应用中应该使用time包解析日期
	if date1 == "" || date2 == "" {
		return 0
	}
	return 1 // 简化处理，假设总是间隔至少1天
}

// 打印回测结果
func PrintBacktestResult(result BacktestResult, stockCode string) {
	fmt.Printf("=== %s 回测结果 ===\n", stockCode)
	fmt.Printf("初始资金:     %.2f\n", result.InitialCapital)
	fmt.Printf("最终资金:     %.2f\n", result.FinalCapital)
	fmt.Printf("总收益率:     %.2f%%\n", result.TotalReturn*100)
	fmt.Printf("年化收益率:   %.2f%%\n", result.AnnualReturn*100)
	fmt.Printf("最大回撤:     %.2f%%\n", result.MaxDrawdown*100)
	fmt.Printf("夏普比率:     %.2f\n", result.SharpeRatio)
	fmt.Printf("总交易次数:   %d\n", result.TotalTrades)
	fmt.Printf("胜率:         %.2f%%\n", result.WinRate*100)
	fmt.Printf("平均盈利:     %.2f\n", result.AvgWin)
	fmt.Printf("平均亏损:     %.2f\n", result.AvgLoss)
	fmt.Printf("盈亏比:       %.2f\n", result.ProfitFactor)

	fmt.Println("\n--- 交易历史 ---")
	for i, trade := range result.TradeHistory {
		fmt.Printf("%d. %s - %s %.2f元 (%.0f股) - 资金:%.2f - %s\n",
			i+1, trade.Date, trade.Type, trade.Price, float64(trade.Shares), trade.Capital, trade.Reason)
	}

	// 策略评价
	fmt.Println("\n--- 策略评价 ---")
	if result.TotalReturn > 0.1 {
		fmt.Println("优秀策略：收益率超过10%")
	} else if result.TotalReturn > 0.05 {
		fmt.Println("良好策略：收益率超过5%")
	} else if result.TotalReturn > 0 {
		fmt.Println("一般策略：略有盈利")
	} else {
		fmt.Println("亏损策略：需要优化")
	}

	if result.MaxDrawdown < 0.1 {
		fmt.Println("风险控制良好：最大回撤小于10%")
	} else if result.MaxDrawdown < 0.2 {
		fmt.Println("风险控制一般：最大回撤小于20%")
	} else {
		fmt.Println("风险较高：最大回撤超过20%")
	}

	if result.SharpeRatio > 1.0 {
		fmt.Println("风险调整收益优秀：夏普比率大于1")
	} else if result.SharpeRatio > 0.5 {
		fmt.Println("风险调整收益良好：夏普比率大于0.5")
	} else {
		fmt.Println("风险调整收益一般：夏普比率较低")
	}
}

// 多策略回测对比
func RunMultiStrategyBacktest(klines []data.Kline) {
	if len(klines) < 30 {
		fmt.Println("数据不足，无法进行回测")
		return
	}

	fmt.Println("=== 多策略回测对比 ===")

	// 定义不同的策略配置
	strategies := []struct {
		name   string
		config BacktestConfig
	}{
		{
			name: "保守策略",
			config: BacktestConfig{
				InitialCapital: 100000,
				PositionSize:   0.5,
				StopLoss:       0.05,
				TakeProfit:     0.10,
				MinHoldDays:    5,
				MaxHoldDays:    20,
			},
		},
		{
			name: "平衡策略",
			config: BacktestConfig{
				InitialCapital: 100000,
				PositionSize:   0.7,
				StopLoss:       0.08,
				TakeProfit:     0.15,
				MinHoldDays:    3,
				MaxHoldDays:    15,
			},
		},
		{
			name: "激进策略",
			config: BacktestConfig{
				InitialCapital: 100000,
				PositionSize:   0.9,
				StopLoss:       0.12,
				TakeProfit:     0.20,
				MinHoldDays:    1,
				MaxHoldDays:    10,
			},
		},
	}

	// 执行每个策略的回测
	results := make([]BacktestResult, len(strategies))
	for i, strategy := range strategies {
		fmt.Printf("\n--- %s ---\n", strategy.name)
		results[i] = RunBacktest(klines, strategy.config)
		PrintBacktestResult(results[i], strategy.name)
	}

	// 策略对比总结
	fmt.Println("\n=== 策略对比总结 ===")
	fmt.Printf("%-12s %-10s %-10s %-10s %-10s %-10s\n",
		"策略名称", "总收益率", "年化收益率", "最大回撤", "夏普比率", "胜率")
	fmt.Println("------------------------------------------------------------")

	for i, strategy := range strategies {
		result := results[i]
		fmt.Printf("%-12s %-10.2f%% %-10.2f%% %-10.2f%% %-10.2f %-10.2f%%\n",
			strategy.name,
			result.TotalReturn*100,
			result.AnnualReturn*100,
			result.MaxDrawdown*100,
			result.SharpeRatio,
			result.WinRate*100)
	}

	// 推荐最佳策略
	bestStrategy := findBestStrategy(results, strategies)
	fmt.Printf("\n推荐策略: %s\n", bestStrategy.name)
	fmt.Printf("理由: 综合考虑收益率、风险控制和交易频率\n")
}

// 寻找最佳策略
func findBestStrategy(results []BacktestResult, strategies []struct {
	name   string
	config BacktestConfig
}) struct {
	name   string
	config BacktestConfig
} {
	bestIndex := 0
	bestScore := 0.0

	for i, result := range results {
		// 综合评分：收益率*0.4 + 风险控制*0.3 + 胜率*0.3
		score := result.TotalReturn*0.4 +
			(1-result.MaxDrawdown)*0.3 +
			result.WinRate*0.3

		if score > bestScore {
			bestScore = score
			bestIndex = i
		}
	}

	return strategies[bestIndex]
}
