# Quantix - 股票量化分析系统

[![Go Version](https://img.shields.io/github/go-mod/go-version/sy-vendor/Quantix)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Issues](https://img.shields.io/github/issues/sy-vendor/Quantix)](https://github.com/sy-vendor/Quantix/issues)
[![Pull Requests](https://img.shields.io/github/issues-pr/sy-vendor/Quantix)](https://github.com/sy-vendor/Quantix/pulls)
[![Stars](https://img.shields.io/github/stars/sy-vendor/Quantix)](https://github.com/sy-vendor/Quantix/stargazers)
[![Forks](https://img.shields.io/github/forks/sy-vendor/Quantix)](https://github.com/sy-vendor/Quantix/network/members)

一个基于Go语言开发的股票量化分析系统，支持A股和美股的技术分析、趋势预测、策略回测和图表可视化。

## 🚀 主要功能

### 1. 数据获取
- **A股数据**: 使用腾讯财经API获取实时行情数据
- **美股数据**: 使用Yahoo Finance API获取历史数据，自动切换数据源
- **本地CSV导入**: 支持通过`--csv`参数导入本地K线数据
- 支持K线数据（开高低收量）

### 2. 技术分析
- **移动平均线**: MA5、MA10、MA20、MA30
- **技术指标**: RSI、MACD、动量、波动率、换手率、布林带、KDJ、WR、CCI、ATR、OBV
- **多因子打分**: 支持自定义因子权重的多因子综合评分与选股
- **技术分析建议**: 基于多指标的综合分析

### 3. 机器学习预测
- **模型支持**: 线性回归、移动平均、技术指标法、决策树、随机森林、集成方法
- **预测内容**: 下一天/周/月价格、趋势、置信度、历史准确率
- **自动特征工程**: 自动提取常用因子作为特征
- **模型融合**: 多模型集成预测

### 4. 风险管理
- **风险指标**: VaR(95/99%)、最大回撤、夏普比率、索提诺比率、卡玛比率、年化波动率、偏度、峰度、下行偏差、捕获率
- **风险评级**: 自动评估高/中/低风险
- **风险建议**: 针对不同风险水平给出操作建议

### 5. 策略回测
- **多策略对比**: 保守、平衡、激进三种策略
- **关键指标**: 收益率、最大回撤、夏普比率、胜率
- **交易历史**: 详细的买卖记录和原因
- **策略评价**: 自动评估策略表现

### 6. 图表可视化
- **K线图**: 价格走势图
- **回测图表**: 资金曲线和交易记录
- **综合分析图**: 技术指标、预测结果和回测对比
- **HTML格式**: 可在浏览器中查看，支持交互

### 7. 多股票对比
- **多股票评分**: 支持多只股票因子打分、风险对比、建议输出
- **自定义因子与权重**: 命令行参数`--factors`、`--weights`灵活配置

## 🧪 测试与开发

- **测试套件**: `test/` 目录下包含全面的单元测试，覆盖技术指标、风险管理、ML预测、多股对比等
- **一键测试**: `./test/run_tests.sh` 支持分模块或全量测试
- **开发建议**: 推荐使用 Go 1.21+，如需扩展新因子/模型，参考`analysis/`目录下相关模块

## 🛠️ 使用方法

### 基本使用
```bash
# 分析单只股票
go run main.go <股票代码> <开始日期> <结束日期>

# 多股票对比（可指定因子和权重）
go run main.go 000001.SZ,600519.SH 2024-01-01 2024-04-01 --factors MA5,RSI,MACD --weights 0.3,0.4,0.3

# 使用本地CSV数据
go run main.go --csv mydata.csv
```

### 查看图表
1. 运行分析后，系统会自动在`charts/`目录生成HTML图表文件
2. 用浏览器打开生成的HTML文件即可查看图表
3. 文件名格式：`股票代码_图表类型_时间戳.html`

### 机器学习预测
- 自动输出多模型预测结果、置信度、趋势、历史准确率
- 支持自定义特征和模型扩展

### 风险管理
- 自动输出多项风险指标和评级
- 支持极端行情和多样化数据

### 测试
```bash
# 运行全部测试
go test ./test -v
# 只运行ML相关测试
./test/run_tests.sh --ml --verbose
```

## 📁 项目结构

```
Quantix/
├── main.go              # 主程序入口
├── data/
│   └── fetch.go         # 数据获取模块
├── analysis/
│   ├── factors.go       # 技术指标计算
│   ├── prediction.go    # 趋势预测
│   ├── ml_prediction.go # 机器学习预测
│   ├── backtest.go      # 回测功能
│   ├── charts.go        # 图表生成
│   ├── print.go         # 结果打印
│   ├── risk.go          # 风险管理
│   └── stock_compare.go # 股票对比
├── charts/              # 生成的图表文件目录
├── test/                # 测试用例与数据
└── go.mod              # Go模块文件
```

## ✅ 已实现功能
- [x] A股/美股/本地CSV数据自动切换
- [x] 多种技术指标与因子计算
- [x] 多因子打分与自定义权重
- [x] 机器学习多模型预测（线性回归、决策树、随机森林、集成）
- [x] 风险管理与多项风险指标
- [x] 策略回测与多策略对比
- [x] 多股票评分与建议
- [x] 交互式HTML图表
- [x] 完整测试套件

## 🚧 待扩展功能
- [ ] 实时数据更新
- [ ] Web界面
- [ ] 参数自动优化
- [ ] 投资组合管理

## 📝 注意事项

1. **数据限制**: 免费API可能有访问频率限制
2. **预测风险**: 历史数据不代表未来表现，投资需谨慎
3. **回测结果**: 回测结果仅供参考，实际交易需要考虑更多因素
4. **浏览器兼容**: 图表需要现代浏览器支持

## 🤝 贡献

欢迎提交Issue和Pull Request来改进这个项目！

---

**免责声明**: 本系统仅供学习和研究使用，不构成投资建议。投资有风险，入市需谨慎。 