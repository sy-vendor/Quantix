# Quantix - 股票量化分析系统

[![Go Version](https://img.shields.io/github/go-mod/go-version/sy-vendor/Quantix)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Issues](https://img.shields.io/github/issues/sy-vendor/Quantix)](https://github.com/sy-vendor/Quantix/issues)
[![Pull Requests](https://img.shields.io/github/issues-pr/sy-vendor/Quantix)](https://github.com/sy-vendor/Quantix/pulls)
[![Stars](https://img.shields.io/github/stars/sy-vendor/Quantix)](https://github.com/sy-vendor/Quantix/stargazers)
[![Forks](https://img.shields.io/github/forks/sy-vendor/Quantix)](https://github.com/sy-vendor/Quantix/network/members)
[![Docker](https://img.shields.io/badge/Docker-Ready-blue.svg)](https://www.docker.com/)
[![Prometheus](https://img.shields.io/badge/Monitoring-Prometheus-orange.svg)](https://prometheus.io/)

一个基于Go语言开发的现代化股票量化分析系统，支持A股和美股的技术分析、趋势预测、策略回测和图表可视化。现已升级为微服务架构，支持Web API、实时监控和容器化部署。

## 🚀 主要功能

### 1. 数据获取
- **A股数据**: 使用腾讯财经API获取实时行情数据
- **美股数据**: 使用Yahoo Finance API获取历史数据，自动切换数据源
- **本地CSV导入**: 支持通过`--csv`参数导入本地K线数据
- **数据缓存**: Redis缓存支持，提高数据获取效率
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
- **模型持久化**: 支持模型保存和加载

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

### 8. Web API服务
- **RESTful API**: 完整的HTTP API接口
- **WebSocket**: 实时数据推送
- **JWT认证**: 安全的API访问控制
- **CORS支持**: 跨域请求支持

### 9. 监控和运维
- **Prometheus监控**: 完整的指标收集
- **Grafana仪表板**: 可视化监控界面
- **结构化日志**: JSON格式日志输出
- **健康检查**: 自动健康状态检测

## 🛠️ 技术架构

### 核心技术栈
- **语言**: Go 1.22
- **Web框架**: Gin
- **数据库**: PostgreSQL
- **缓存**: Redis
- **监控**: Prometheus + Grafana
- **容器化**: Docker + Docker Compose
- **机器学习**: golearn

### 系统架构
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Client    │    │   Mobile App    │    │   API Client    │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                    ┌─────────────▼─────────────┐
                    │      Quantix API          │
                    │    (Gin Web Server)       │
                    └─────────────┬─────────────┘
                                  │
          ┌───────────────────────┼───────────────────────┐
          │                       │                       │
┌─────────▼─────────┐  ┌─────────▼─────────┐  ┌─────────▼─────────┐
│   Analysis Core   │  │   ML Engine       │  │   Data Service    │
│   - 技术指标       │  │   - 预测模型       │  │   - 数据获取       │
│   - 风险管理       │  │   - 模型训练       │  │   - 数据缓存       │
│   - 策略回测       │  │   - 模型评估       │  │   - 数据清洗       │
└─────────┬─────────┘  └─────────┬─────────┘  └─────────┬─────────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                    ┌─────────────▼─────────────┐
                    │      Data Layer          │
                    │  ┌─────────┬─────────┐   │
                    │  │PostgreSQL│ Redis  │   │
                    │  └─────────┴─────────┘   │
                    └─────────────────────────┘
```

## 🚀 快速开始

### 1. 使用Docker（推荐）

```bash
# 克隆项目
git clone https://github.com/sy-vendor/Quantix.git
cd Quantix

# 启动所有服务
make docker-run

# 查看服务状态
docker-compose ps

# 访问服务
# API: http://localhost:8080
# Grafana: http://localhost:3000 (admin/admin)
# Prometheus: http://localhost:9090
```

### 2. 本地开发

```bash
# 安装依赖
make install

# 运行测试
make test

# 启动应用
make run

# 或者直接运行
go run main.go AAPL 2024-01-01 2024-04-01
```

### 3. API使用示例

```bash
# 获取股票分析
curl "http://localhost:8080/api/v1/stock/AAPL?start=2024-01-01&end=2024-04-01"

# 股票对比
curl -X POST "http://localhost:8080/api/v1/stock/compare" \
  -H "Content-Type: application/json" \
  -d '{
    "codes": ["AAPL", "MSFT"],
    "start": "2024-01-01",
    "end": "2024-04-01",
    "factors": ["MA5", "RSI", "MACD"],
    "weights": [0.3, 0.4, 0.3]
  }'

# 获取预测
curl "http://localhost:8080/api/v1/stock/AAPL/predict"

# 获取风险指标
curl "http://localhost:8080/api/v1/stock/AAPL/risk"
```

## 📁 项目结构

```
Quantix/
├── main.go                 # 主程序入口
├── config/
│   └── config.go          # 配置管理
├── logger/
│   └── logger.go          # 日志系统
├── api/
│   └── server.go          # Web API服务器
├── cache/
│   └── redis.go           # Redis缓存
├── monitoring/
│   └── metrics.go         # Prometheus指标
├── data/
│   └── fetch.go           # 数据获取模块
├── analysis/
│   ├── factors.go         # 技术指标计算
│   ├── prediction.go      # 趋势预测
│   ├── ml_prediction.go   # 机器学习预测
│   ├── backtest.go        # 回测功能
│   ├── charts.go          # 图表生成
│   ├── print.go           # 结果打印
│   ├── risk.go            # 风险管理
│   └── stock_compare.go   # 股票对比
├── charts/                # 生成的图表文件目录
├── models/                # 机器学习模型目录
├── uploads/               # 上传文件目录
├── test/                  # 测试用例与数据
├── scripts/               # 数据库脚本
├── monitoring/            # 监控配置
│   ├── prometheus.yml
│   └── grafana/
├── Dockerfile             # Docker配置
├── docker-compose.yml     # Docker Compose配置
├── Makefile               # 构建工具
├── config.yaml            # 配置文件
├── go.mod                 # Go模块文件
└── README.md              # 项目文档
```

## 🧪 测试与开发

### 运行测试
```bash
# 运行所有测试
make test

# 运行特定测试
make test-factors    # 技术指标测试
make test-ml         # 机器学习测试
make test-risk       # 风险管理测试

# 性能测试
make benchmark

# 覆盖率测试
make coverage
```

### 代码质量
```bash
# 代码格式化
make format

# 安全检查
make security
```

## 📊 监控和运维

### 监控指标
- **应用指标**: 请求数、响应时间、错误率
- **业务指标**: 数据获取次数、预测准确率
- **系统指标**: CPU、内存、磁盘使用率
- **缓存指标**: 命中率、连接数

### 日志管理
- **结构化日志**: JSON格式，便于分析
- **日志级别**: DEBUG、INFO、WARN、ERROR
- **日志输出**: 支持文件和控制台输出

### 健康检查
```bash
# 检查API健康状态
curl http://localhost:8080/health

# 检查Docker容器状态
docker-compose ps
```

## 🔧 配置说明

### 环境变量
```bash
# 服务器配置
QUANTIX_SERVER_PORT=8080

# 数据库配置
QUANTIX_DATABASE_HOST=localhost
QUANTIX_DATABASE_PORT=5432
QUANTIX_DATABASE_USER=quantix
QUANTIX_DATABASE_PASSWORD=quantix123

# Redis配置
QUANTIX_REDIS_HOST=localhost
QUANTIX_REDIS_PORT=6379

# 日志配置
QUANTIX_LOG_LEVEL=info
QUANTIX_LOG_FORMAT=json
```

### 配置文件
项目使用YAML配置文件，支持环境变量覆盖。详细配置请参考 `config.yaml`。

## 🚀 部署指南

### 生产环境部署
```bash
# 1. 构建生产镜像
make docker-build

# 2. 配置生产环境变量
export QUANTIX_ENV=production

# 3. 启动生产服务
docker-compose -f docker-compose.prod.yml up -d

# 4. 检查服务状态
docker-compose ps
```

### 备份和恢复
```bash
# 备份数据
make backup

# 恢复数据
make restore
```

## 🤝 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

### 开发规范
- 遵循 Go 代码规范
- 添加单元测试
- 更新文档
- 运行代码检查

## 📝 更新日志

### v2.0.0 (2025-06-27)
- ✨ 新增Web API服务
- ✨ 新增Docker容器化支持
- ✨ 新增Prometheus监控
- ✨ 新增Redis缓存
- ✨ 新增结构化日志
- ✨ 新增配置管理
- 🔧 升级到Go 1.22
- 🔧 优化性能和稳定性

### v1.0.0 (2023-01-23)
- 🎉 初始版本发布
- ✨ 基础量化分析功能
- ✨ 技术指标计算
- ✨ 机器学习预测
- ✨ 风险管理
- ✨ 策略回测

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## ⚠️ 免责声明

本系统仅供学习和研究使用，不构成投资建议。投资有风险，入市需谨慎。

---

**Quantix** - 让量化投资更简单、更智能！ 