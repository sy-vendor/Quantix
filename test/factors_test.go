package analysis_test

import (
	"Quantix/analysis"
	"Quantix/data"
	"testing"
)

func TestTechnicalIndicators(t *testing.T) {
	t.Log("=== 技术指标测试 ===")

	klines := createTestData()
	if len(klines) < 30 {
		t.Fatal("测试数据不足")
	}

	factors := analysis.CalcFactors(klines)
	if len(factors) == 0 {
		t.Fatal("技术指标计算失败")
	}

	latest := factors[len(factors)-1]

	// 测试RSI
	t.Run("RSI测试", func(t *testing.T) {
		if latest.RSI < 0 || latest.RSI > 100 {
			t.Errorf("RSI值超出范围 [0,100]: %.2f", latest.RSI)
		}
		t.Logf("RSI: %.2f", latest.RSI)
	})

	// 测试MACD
	t.Run("MACD测试", func(t *testing.T) {
		t.Logf("MACD: %.4f", latest.MACD)
		// MACD可以是正值或负值，不做范围限制
	})

	// 测试布林带
	t.Run("布林带测试", func(t *testing.T) {
		if latest.BBUpper <= latest.BBLower {
			t.Errorf("布林带上轨应该大于下轨: 上轨=%.2f, 下轨=%.2f", latest.BBUpper, latest.BBLower)
		}
		t.Logf("布林带上轨: %.2f, 下轨: %.2f", latest.BBUpper, latest.BBLower)
	})

	// 测试KDJ
	t.Run("KDJ测试", func(t *testing.T) {
		if latest.KDJ_K < 0 || latest.KDJ_K > 100 {
			t.Errorf("KDJ-K值超出范围 [0,100]: %.2f", latest.KDJ_K)
		}
		if latest.KDJ_D < 0 || latest.KDJ_D > 100 {
			t.Errorf("KDJ-D值超出范围 [0,100]: %.2f", latest.KDJ_D)
		}
		t.Logf("KDJ-K: %.2f, KDJ-D: %.2f, KDJ-J: %.2f", latest.KDJ_K, latest.KDJ_D, latest.KDJ_J)
	})

	// 测试威廉指标
	t.Run("威廉指标测试", func(t *testing.T) {
		if latest.WR > 0 || latest.WR < -100 {
			t.Errorf("威廉指标值超出范围 [-100,0]: %.2f", latest.WR)
		}
		t.Logf("威廉指标: %.2f", latest.WR)
	})

	// 测试CCI
	t.Run("CCI测试", func(t *testing.T) {
		t.Logf("CCI: %.2f", latest.CCI)
		// CCI可以是任意值，不做范围限制
	})

	// 测试ATR
	t.Run("ATR测试", func(t *testing.T) {
		if latest.ATR <= 0 {
			t.Errorf("ATR应该大于0: %.4f", latest.ATR)
		}
		t.Logf("ATR: %.4f", latest.ATR)
	})

	// 测试OBV
	t.Run("OBV测试", func(t *testing.T) {
		t.Logf("OBV: %.2f", latest.OBV)
		// OBV可以是任意值
	})
}

func TestMovingAverages(t *testing.T) {
	t.Log("=== 移动平均线测试 ===")

	klines := createTestData()
	factors := analysis.CalcFactors(klines)

	if len(factors) == 0 {
		t.Fatal("技术指标计算失败")
	}

	latest := factors[len(factors)-1]

	// 测试各种移动平均线
	t.Run("MA5测试", func(t *testing.T) {
		if latest.MA5 <= 0 {
			t.Errorf("MA5应该大于0: %.2f", latest.MA5)
		}
		t.Logf("MA5: %.2f", latest.MA5)
	})

	t.Run("MA10测试", func(t *testing.T) {
		if latest.MA10 <= 0 {
			t.Errorf("MA10应该大于0: %.2f", latest.MA10)
		}
		t.Logf("MA10: %.2f", latest.MA10)
	})

	t.Run("MA20测试", func(t *testing.T) {
		if latest.MA20 <= 0 {
			t.Errorf("MA20应该大于0: %.2f", latest.MA20)
		}
		t.Logf("MA20: %.2f", latest.MA20)
	})

	t.Run("MA30测试", func(t *testing.T) {
		if latest.MA30 <= 0 {
			t.Errorf("MA30应该大于0: %.2f", latest.MA30)
		}
		t.Logf("MA30: %.2f", latest.MA30)
	})
}

func TestVolumeIndicators(t *testing.T) {
	t.Log("=== 成交量指标测试 ===")

	klines := createTestData()
	factors := analysis.CalcFactors(klines)

	if len(factors) == 0 {
		t.Fatal("技术指标计算失败")
	}

	latest := factors[len(factors)-1]

	// 测试成交量相关指标
	t.Run("成交量测试", func(t *testing.T) {
		if latest.Volume <= 0 {
			t.Errorf("成交量应该大于0: %.2f", latest.Volume)
		}
		t.Logf("成交量: %.2f", latest.Volume)
	})

	t.Run("换手率测试", func(t *testing.T) {
		if latest.Turnover < 0 {
			t.Errorf("换手率不应该为负: %.2f", latest.Turnover)
		}
		t.Logf("换手率: %.2f", latest.Turnover)
	})
}

func TestMomentumIndicators(t *testing.T) {
	t.Log("=== 动量指标测试 ===")

	klines := createTestData()
	factors := analysis.CalcFactors(klines)

	if len(factors) == 0 {
		t.Fatal("技术指标计算失败")
	}

	latest := factors[len(factors)-1]

	// 测试动量指标
	t.Run("动量测试", func(t *testing.T) {
		t.Logf("动量: %.4f", latest.Momentum)
		// 动量可以是正值或负值
	})
}

func TestVolatilityIndicators(t *testing.T) {
	t.Log("=== 波动率指标测试 ===")

	klines := createTestData()
	factors := analysis.CalcFactors(klines)

	if len(factors) == 0 {
		t.Fatal("技术指标计算失败")
	}

	latest := factors[len(factors)-1]

	// 测试波动率指标
	t.Run("波动率测试", func(t *testing.T) {
		if latest.Volatility < 0 {
			t.Errorf("波动率不应该为负: %.2f%%", latest.Volatility)
		}
		t.Logf("波动率: %.2f%%", latest.Volatility)
	})
}

func TestDataConsistency(t *testing.T) {
	t.Log("=== 数据一致性测试 ===")

	testCases := []struct {
		name   string
		klines []data.Kline
	}{
		{"正常数据", createTestData()},
		{"低波动数据", createLowVolatilityData()},
		{"高波动数据", createHighVolatilityData()},
		{"趋势数据", createTrendingData()},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.klines) < 20 {
				t.Skip("数据不足，跳过测试")
			}

			factors := analysis.CalcFactors(tc.klines)
			if len(factors) == 0 {
				t.Fatal("技术指标计算失败")
			}

			// 检查数据一致性
			for i, factor := range factors {
				if factor.RSI < 0 || factor.RSI > 100 {
					t.Errorf("第%d个因子RSI值异常: %.2f", i, factor.RSI)
				}
				if i >= 4 && factor.MA5 <= 0 {
					t.Errorf("第%d个因子MA5值异常: %.2f", i, factor.MA5)
				}
			}

			t.Logf("%s: 生成了%d个技术因子", tc.name, len(factors))
		})
	}
}
