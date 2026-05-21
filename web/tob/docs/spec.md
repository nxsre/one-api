# 需求说明

## 总体目标

基于当前one-api项目的API，开发一个toB端的用户友好页面，给toB的客户使用，只做页面样式的修改，业务逻辑还是one-api的

## 功能

- 登录 (MFA)
- 概览(总览)
- 模型广场(渠道)
- 可观测性
  - 用量统计(平台财务报表)
  - 日志(操作日志)
- 管理
  - API KEY(令牌管理)
  - 个人设置(系统设置)

## 技术选型

- React 18 + Vite 6 + React Router
- Docker + nginx
- 实现目录：`web/tob/`（见同目录 `README.md`）

## 已搭建（v0.1）

- [x] 项目脚手架、主题 CSS（参考 TokenHub mockup）
- [x] 侧栏菜单与路由
- [x] 登录 + MFA（对接 one-api `/api/user/login`）
- [x] 概览页对接 `/api/user/dashboard`
- [ ] 其余页面：迁移 default 业务组件与 mockup 完整 UI
