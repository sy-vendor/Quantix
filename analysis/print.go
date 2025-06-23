package analysis

import "fmt"

func PrintAnalysis(f Factors) {
	fmt.Println("=== 量化因子分析 ===")
	fmt.Println("--- 移动平均线 ---")
	fmt.Printf("5日均线:  %.2f\n", f.MA5)
	fmt.Printf("10日均线: %.2f\n", f.MA10)
	fmt.Printf("20日均线: %.2f\n", f.MA20)

	fmt.Println("\n--- 价格指标 ---")
	fmt.Printf("动量:     %.2f\n", f.Momentum)
	fmt.Printf("波动率:   %.4f\n", f.Volatility)

	fmt.Println("\n--- 技术指标 ---")
	fmt.Printf("换手率:   %.2f (相对20日均量)\n", f.Turnover)
	fmt.Printf("RSI(14):  %.2f\n", f.RSI)
	fmt.Printf("MACD:     %.4f\n", f.MACD)
	fmt.Printf("MACD信号: %.4f\n", f.MACDSignal)
	fmt.Printf("MACD柱:   %.4f\n", f.MACDHist)

	// 添加简单的技术分析建议
	fmt.Println("\n--- 技术分析建议 ---")
	if f.RSI > 70 {
		fmt.Println("RSI > 70: 可能超买，注意回调风险")
	} else if f.RSI < 30 {
		fmt.Println("RSI < 30: 可能超卖，关注反弹机会")
	} else {
		fmt.Println("RSI正常区间: 价格相对稳定")
	}

	if f.MACD > f.MACDSignal {
		fmt.Println("MACD > 信号线: 短期趋势向上")
	} else {
		fmt.Println("MACD < 信号线: 短期趋势向下")
	}

	if f.Turnover > 1.5 {
		fmt.Println("换手率较高: 交易活跃，注意量价配合")
	} else if f.Turnover < 0.5 {
		fmt.Println("换手率较低: 交易清淡，等待放量")
	} else {
		fmt.Println("换手率正常: 交易量适中")
	}
}
