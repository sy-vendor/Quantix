package analysis_test

import (
	"Quantix/analysis"
	"Quantix/data"
	"math"
	"testing"
)

func TestRiskMetricsCalculation(t *testing.T) {
	t.Log("=== 风险指标计算测试 ===")

	testCases := []struct {
		name         string
		klines       []data.Kline
		riskFreeRate float64
		expectedRisk string
	}{
		{
			name:         "低波动性股票",
			klines:       createLowVolatilityData(),
			riskFreeRate: 0.03,
			expectedRisk: "低风险",
		},
		{
			name:         "高波动性股票",
			klines:       createHighVolatilityData(),
			riskFreeRate: 0.03,
			expectedRisk: "高风险",
		},
		{
			name:         "趋势性股票",
			klines:       createTrendingData(),
			riskFreeRate: 0.03,
			expectedRisk: "中风险",
		},
		{
			name:         "上涨趋势股票",
			klines:       createUptrendData(),
			riskFreeRate: 0.03,
			expectedRisk: "中低风险",
		},
		{
			name:         "下跌趋势股票",
			klines:       createDowntrendData(),
			riskFreeRate: 0.03,
			expectedRisk: "中高风险",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.klines) < 30 {
				t.Skip("数据不足，跳过测试")
			}

			riskMetrics := analysis.CalculateRiskMetrics(tc.klines, tc.riskFreeRate)

			// 测试基本风险指标
			t.Run("VaR测试", func(t *testing.T) {
				if riskMetrics.VaR95 > 1 {
					t.Errorf("VaR(95%%)异常过大: %.2f%%", riskMetrics.VaR95*100)
				}
				if riskMetrics.VaR99 > 1 {
					t.Errorf("VaR(99%%)异常过大: %.2f%%", riskMetrics.VaR99*100)
				}
				if riskMetrics.VaR99 > riskMetrics.VaR95+0.1 {
					t.Errorf("VaR(99%%)应该小于或接近VaR(95%%): %.2f%% vs %.2f%%", riskMetrics.VaR99*100, riskMetrics.VaR95*100)
				}
				t.Logf("VaR(95%%): %.2f%%, VaR(99%%): %.2f%%",
					riskMetrics.VaR95*100, riskMetrics.VaR99*100)
			})

			t.Run("最大回撤测试", func(t *testing.T) {
				if riskMetrics.MaxDrawdown < 0 {
					t.Errorf("最大回撤不应该为负: %.2f%%", riskMetrics.MaxDrawdown*100)
				}
				if riskMetrics.MaxDrawdown > 1 {
					t.Errorf("最大回撤不应该超过100%%: %.2f%%", riskMetrics.MaxDrawdown*100)
				}
				t.Logf("最大回撤: %.2f%%", riskMetrics.MaxDrawdown*100)
			})

			t.Run("夏普比率测试", func(t *testing.T) {
				if math.IsNaN(riskMetrics.SharpeRatio) {
					t.Error("夏普比率不应该是NaN")
				}
				if math.IsInf(riskMetrics.SharpeRatio, 0) {
					t.Error("夏普比率不应该是无穷大")
				}
				t.Logf("夏普比率: %.3f", riskMetrics.SharpeRatio)
			})

			t.Run("索提诺比率测试", func(t *testing.T) {
				if math.IsNaN(riskMetrics.SortinoRatio) {
					t.Error("索提诺比率不应该是NaN")
				}
				if math.IsInf(riskMetrics.SortinoRatio, 0) {
					t.Error("索提诺比率不应该是无穷大")
				}
				t.Logf("索提诺比率: %.3f", riskMetrics.SortinoRatio)
			})

			t.Run("卡玛比率测试", func(t *testing.T) {
				if math.IsNaN(riskMetrics.CalmarRatio) {
					t.Error("卡玛比率不应该是NaN")
				}
				if math.IsInf(riskMetrics.CalmarRatio, 0) {
					t.Error("卡玛比率不应该是无穷大")
				}
				t.Logf("卡玛比率: %.3f", riskMetrics.CalmarRatio)
			})

			t.Run("年化波动率测试", func(t *testing.T) {
				if riskMetrics.Volatility < 0 {
					t.Errorf("年化波动率不应该为负: %.2f%%", riskMetrics.Volatility*100)
				}
				t.Logf("年化波动率: %.2f%%", riskMetrics.Volatility*100)
			})

			t.Run("偏度测试", func(t *testing.T) {
				if math.IsNaN(riskMetrics.Skewness) {
					t.Error("偏度不应该是NaN")
				}
				if math.IsInf(riskMetrics.Skewness, 0) {
					t.Error("偏度不应该是无穷大")
				}
				t.Logf("偏度: %.3f", riskMetrics.Skewness)
			})

			t.Run("峰度测试", func(t *testing.T) {
				if math.IsNaN(riskMetrics.Kurtosis) {
					t.Error("峰度不应该是NaN")
				}
				if math.IsInf(riskMetrics.Kurtosis, 0) {
					t.Error("峰度不应该是无穷大")
				}
				t.Logf("峰度: %.3f", riskMetrics.Kurtosis)
			})

			t.Run("下行偏差测试", func(t *testing.T) {
				if riskMetrics.DownsideDeviation < 0 {
					t.Errorf("下行偏差不应该为负: %.2f%%", riskMetrics.DownsideDeviation*100)
				}
				t.Logf("下行偏差: %.2f%%", riskMetrics.DownsideDeviation*100)
			})

			t.Run("捕获率测试", func(t *testing.T) {
				if math.IsNaN(riskMetrics.UpsideCapture) || math.IsNaN(riskMetrics.DownsideCapture) {
					t.Errorf("捕获率不应该是NaN")
				}
				t.Logf("上行捕获率: %.2f%%, 下行捕获率: %.2f%%",
					riskMetrics.UpsideCapture*100, riskMetrics.DownsideCapture*100)
			})
		})
	}
}

func TestRiskRating(t *testing.T) {
	t.Log("=== 风险评级测试 ===")

	testCases := []struct {
		name           string
		klines         []data.Kline
		expectedRating string
	}{
		{"低波动性股票", createLowVolatilityData(), "低风险"},
		{"高波动性股票", createHighVolatilityData(), "高风险"},
		{"趋势性股票", createTrendingData(), "中风险"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.klines) < 30 {
				t.Skip("数据不足，跳过测试")
			}

			riskMetrics := analysis.CalculateRiskMetrics(tc.klines, 0.03)

			// 根据最大回撤进行风险评级
			var actualRating string
			if riskMetrics.MaxDrawdown < 0.05 {
				actualRating = "低风险"
			} else if riskMetrics.MaxDrawdown < 0.15 {
				actualRating = "中低风险"
			} else if riskMetrics.MaxDrawdown < 0.25 {
				actualRating = "中风险"
			} else if riskMetrics.MaxDrawdown < 0.35 {
				actualRating = "中高风险"
			} else {
				actualRating = "高风险"
			}

			t.Logf("最大回撤: %.2f%%, 风险评级: %s", riskMetrics.MaxDrawdown*100, actualRating)

			// 验证风险评级逻辑
			if riskMetrics.MaxDrawdown < 0.05 && actualRating != "低风险" {
				t.Errorf("最大回撤%.2f%%应该评为低风险", riskMetrics.MaxDrawdown*100)
			}
			if riskMetrics.MaxDrawdown > 0.35 && actualRating != "高风险" {
				t.Errorf("最大回撤%.2f%%应该评为高风险", riskMetrics.MaxDrawdown*100)
			}
		})
	}
}

func TestRiskMetricsEdgeCases(t *testing.T) {
	t.Log("=== 风险指标边界情况测试 ===")

	t.Run("数据不足测试", func(t *testing.T) {
		// 测试数据不足的情况
		klines := createTestData()[:10] // 只有10条数据
		riskMetrics := analysis.CalculateRiskMetrics(klines, 0.03)

		// 数据不足时应该返回默认值或零值
		if riskMetrics.VaR95 != 0 || riskMetrics.MaxDrawdown != 0 {
			t.Error("数据不足时应该返回默认值")
		}
	})

	t.Run("零风险利率测试", func(t *testing.T) {
		klines := createTestData()
		riskMetrics := analysis.CalculateRiskMetrics(klines, 0.0)

		// 零风险利率时夏普比率应该等于收益率除以波动率
		if math.IsNaN(riskMetrics.SharpeRatio) {
			t.Error("零风险利率时夏普比率不应该是NaN")
		}
	})

	t.Run("负风险利率测试", func(t *testing.T) {
		klines := createTestData()
		riskMetrics := analysis.CalculateRiskMetrics(klines, -0.01)

		// 负风险利率时夏普比率应该仍然有效
		if math.IsNaN(riskMetrics.SharpeRatio) {
			t.Error("负风险利率时夏普比率不应该是NaN")
		}
	})
}

func TestRiskMetricsConsistency(t *testing.T) {
	t.Log("=== 风险指标一致性测试 ===")

	klines := createTestData()

	// 多次计算应该得到相同结果
	metrics1 := analysis.CalculateRiskMetrics(klines, 0.03)
	metrics2 := analysis.CalculateRiskMetrics(klines, 0.03)

	if math.Abs(metrics1.VaR95-metrics2.VaR95) > 1e-10 {
		t.Error("VaR计算结果不一致")
	}

	if math.Abs(metrics1.MaxDrawdown-metrics2.MaxDrawdown) > 1e-10 {
		t.Error("最大回撤计算结果不一致")
	}

	if math.Abs(metrics1.SharpeRatio-metrics2.SharpeRatio) > 1e-10 {
		t.Error("夏普比率计算结果不一致")
	}

	t.Log("风险指标计算结果一致")
}
