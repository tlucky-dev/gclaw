# GCLaw 命令快速参考

## 一行命令速查表

```bash
# ========== 基础命令 ==========
./gclaw -version                          # 查看版本
./gclaw -help                             # 查看帮助

# ========== 配置初始化 ==========
./gclaw -init openai                      # 创建 OpenAI 配置
./gclaw -init modelscope                  # 创建 ModelScope 配置
./gclaw -init azure                       # 创建 Azure 配置

# ========== 运行模式 ==========
./gclaw -i -config config.json            # 交互模式
echo "任务" | ./gclaw -config config.json # 单次执行
./gclaw -feishu -config config.json       # 飞书模式

# ========== 参数覆盖 ==========
./gclaw -config config.json -model gpt-4                    # 临时换模型
./gclaw -config config.json -sandbox-level strict           # 严格沙箱
./gclaw -config config.json -sandbox-dryrun                 # 干跑模式
./gclaw -config config.json -api-key sk-xxx                 # 临时 API Key
./gclaw -config config.json -max-iterations 5               # 限制迭代次数

# ========== ModelScope 专用 ==========
./gclaw -provider modelscope -model qwen-max -config config.json
```

## 完整文档

详细使用指南请查看：[docs/CLI_GUIDE.md](docs/CLI_GUIDE.md)
