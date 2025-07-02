# Quantix — DeepSeek AI 智能股票分析工具

> 🚀 基于 DeepSeek API 的智能股票分析，支持深度推理与联网搜索，批量分析、定时任务、历史、多语言、PDF导出、邮件/IM推送，助你高效洞察市场！

---

## ✨ 项目亮点

- **DeepSeek AI 智能分析**：支持"深度思考"与"联网搜索"两大模式
- **批量分析**：支持多个股票代码（逗号分隔），主菜单和 CLI 参数均可批量分析
- **定时任务**：支持 `--schedule` 参数，自动定时批量分析，支持分钟、小时、每日等周期
- **一键导出**：支持导出 Markdown、HTML、PDF 格式报告，便于归档和分享
- **邮件/IM 推送**：分析结果可自动发送到邮箱、钉钉/企业微信等
- **主菜单循环体验**：分析完毕后可直接在主菜单继续分析、查历史、查详情或退出
- **命令行参数与交互模式共存**：支持全参数自动化，也支持全交互体验
- **多维度预测**：技术面、基本面、资金面、行业对比、情绪分析等
- **自定义输出**：结构化表格、要点、长文、摘要，满足不同需求
- **置信度与风险偏好**：可选置信度说明，支持保守/激进/风险/机会导向
- **联网搜索内容可选**：新闻、研报、公告、论坛，信息更全面
- **多语言支持**：支持中文/英文分析
- **历史记录管理**：每次分析自动保存，支持历史检索与复用
- **极简依赖**：仅需 Go 1.22+，无需本地行情数据
- **项目分层结构**：主入口 main.go，AI分析/导出/推送/历史等逻辑在 analysis/ 目录

---

## 🖥️ 快速开始

1. **安装 Go 1.22 及以上版本**
2. **获取 DeepSeek API Key**  
   👉 [DeepSeek 官网](https://platform.deepseek.com/)
3. **如需 PDF 导出，请先安装 [wkhtmltopdf](https://wkhtmltopdf.org/downloads.html)**
4. **运行项目（推荐主菜单模式）**
   ```bash
   go run main.go
   ```
5. **主菜单支持如下指令**：
   - `new` 或 `1`：新建AI分析（支持批量，股票代码用逗号分隔）
   - `history` 或 `2`：查看历史记录列表
   - `show <文件名>` 或 `3 <文件名>`：查看指定历史分析
   - `exit` 或 `4`：退出程序

6. **命令行参数模式（适合自动化/脚本/批量/定时/推送）**
   ```bash
   # 批量分析多个股票并导出 PDF/Markdown/HTML
   go run main.go --apikey sk-xxx --model deepseek-chat --stock AAPL,MSFT,GOOG --export md,html,pdf ...

   # 定时任务：每小时自动分析并邮件推送 PDF 附件
   go run main.go --apikey sk-xxx --model deepseek-chat --stock AAPL,MSFT --schedule 1h --export pdf --email user@example.com --smtp-server smtp.example.com --smtp-port 465 --smtp-user user@example.com --smtp-pass yourpass ...

   # 钉钉/企业微信 IM 推送
   go run main.go --apikey ... --model ... --stock ... --webhook https://oapi.dingtalk.com/robot/send?access_token=xxx ...

   # 查看历史
   go run main.go --history
   # 查看指定历史
   go run main.go --show 600036-2025-07-02-164939.json
   ```

---

## 🛠️ 功能详解

| 功能             | 说明                                                                 |
|------------------|----------------------------------------------------------------------|
| 批量分析         | 支持多个股票代码（逗号分隔），批量生成报告                            |
| 定时任务         | --schedule 支持 10m、1h、daily 等周期自动分析                         |
| 一键导出         | --export 支持 md、html、pdf 格式报告                                  |
| 邮件推送         | --email、--smtp-server、--smtp-user、--smtp-pass 支持自动邮件发送      |
| IM推送           | --webhook 支持钉钉/企业微信机器人自动推送                            |
| 预测周期         | 1天、1周、1月、3月、半年、1年（可多选）                              |
| 分析维度         | 技术面、基本面、资金面、行业对比、情绪分析（可多选）                 |
| 输出格式         | 结构化表格、要点、详细长文、摘要                                     |
| 置信度说明       | 可选每个预测结论都要置信度/概率区间                                   |
| 风险/机会偏好    | 保守、激进、风险为主、机会为主                                       |
| 联网搜索内容范围 | 新闻、研报、公告、论坛（可多选，仅联网模式下生效）                   |
| 多语言           | 支持中文（zh）和英文（en）                                           |
| 历史记录         | 自动保存分析参数和AI输出，支持检索与复用                              |
| 项目结构         | main.go 入口，analysis/ai.go（AI分析）、analysis/export.go（导出）、analysis/email.go（邮件）、analysis/webhook.go（IM）、analysis/history.go（历史） |

---

## 💡 主菜单与批量/定时/推送示例

```
=== Quantix 主菜单 ===
1. new      - 新建AI分析（支持批量，股票代码用逗号分隔）
2. history  - 查看历史记录列表
3. show <文件名> - 查看指定历史分析
4. exit     - 退出程序
请输入指令: new
请输入股票代码（可批量，逗号分隔）: AAPL,MSFT,GOOG
请选择导出格式: md,html,pdf
如需邮件推送请输入收件人邮箱: user@example.com
SMTP服务器: smtp.example.com
SMTP端口(默认465): 465
SMTP用户名: user@example.com
SMTP密码: yourpass
如需IM推送请输入Webhook地址: https://oapi.dingtalk.com/robot/send?access_token=xxx
...（依次生成、导出、推送每只股票的分析报告）...

# CLI 批量分析并导出
$ go run main.go --apikey ... --model ... --stock AAPL,MSFT,GOOG --export md,html,pdf ...

# CLI 定时任务+邮件推送
$ go run main.go --apikey ... --model ... --stock ... --schedule 1h --export pdf --email user@example.com --smtp-server smtp.example.com --smtp-user user@example.com --smtp-pass yourpass ...

# CLI IM推送
$ go run main.go --apikey ... --model ... --stock ... --webhook https://oapi.dingtalk.com/robot/send?access_token=xxx ...
```

---

## 📢 免责声明

> **本系统仅供学习和研究使用，不构成任何投资建议或买卖依据。股市有风险，投资需谨慎。请用户自行甄别分析结果并独立决策，因使用本系统造成的任何后果，开发者不承担任何责任。**

---