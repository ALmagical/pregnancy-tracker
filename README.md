# 孕期管理小程序（孕育时光）

本仓库当前以产品/研发规划文档为主，后续将落地为**原生微信小程序**（WXML/WXSS/JS）。

## 文档

- 产品设计文档：`docs/孕期管理小程序产品设计文档_详细版.md`
- 研发任务规划（含测试与 Git 策略）：`docs/研发任务规划_测试_Git策略.md`

## 推荐研发顺序（摘要）

1. Milestone 0：工程初始化 + 请求/错误/状态底座 + 埋点/合规底座
2. Milestone 1（P0）：孕期信息与首页 → 产检与报告 → 体重/胎动/宫缩/清单
3. Milestone 2（P1）：知识 → AI → 数据导出

更详细内容见 `docs/研发任务规划_测试_Git策略.md`。

## Git 使用建议（摘要）

- 分支：`main`（稳定）+ `feature/<scope>`（按模块）
- 提测：可引入 `release/<version>` 分支管理提测修复
- 提交：建议使用 Conventional Commits（`feat/fix/docs/chore/test`）
