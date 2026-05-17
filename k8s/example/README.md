# 示例清单说明

部署前替换 `secret-config.yaml` 中的 DSN、密钥与镜像占位符，勿将真实秘密提交到 Git。

- **就绪 / 存活探针**：默认使用 **HTTP** 访问 `/api/status`（`deployment-primary.yaml`、`deployment-worker.yaml`）。若在 Pod 内启用嵌入式 TLS，请按文件内注释切换为 **`port: https`、`scheme: HTTPS`**，并同步 `containerPort`、Secret 中的 `https_port` 与 **Service** 的 `targetPort`。
- **完整说明与 YAML 片段**：仓库 **`docs/kubernetes-deployment.md`** 第 4 节（健康检查）。
