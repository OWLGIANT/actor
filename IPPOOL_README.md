# IP池生成功能使用说明

## 功能概述

从 beasttool 迁移的 IP池生成功能，支持三种使用方式：

1. **独立命令行工具** - 手动执行生成 IP池
2. **定时任务调度器** - 自动定期更新 IP池
3. **代码集成** - 在你的程序中调用相关函数

## 使用方式

### 1. 独立命令行工具

位置：`tools/ippool/ippool`

```bash
# 查看帮助
./tools/ippool/ippool -h

# 生成 IP池
./tools/ippool/ippool
```

功能：获取本机所有私有IP，查询对应的公网IP，并保存到 `ipPool_v1.json` 文件。

### 2. 定时任务调度器

位置：`cmd/ippool_schedule/ippool_schedule`

```bash
# 运行定时任务调度器
./cmd/ippool_schedule/ippool_schedule
```

功能：
- 启动时立即执行一次 IP池生成
- 之后每2小时自动执行一次
- 按 Ctrl+C 退出

### 3. 在代码中集成

#### 方式 A：直接调用生成函数

```go
import "actor/helper"

func main() {
    // 手动生成 IP池
    if err := helper.GenIpPool(); err != nil {
        log.Errorf("生成IP池失败: %s", err.Error())
    }
}
```

#### 方式 B：启动定时任务调度器

```go
import "actor/schedule"

func main() {
    // 初始化定时任务调度器
    // 会立即执行一次，然后每2小时自动执行一次
    schedule.InitSchedule()

    // 你的其他代码...
}
```

## 文件说明

- `helper/ip.go` - IP池核心功能实现
  - `GetClientIp()` - 获取本机私有IP
  - `GenIpPool()` - 生成IP池并保存到文件
  - `GetIpPool()` - 读取IP池文件

- `schedule/gen-ippool.go` - IP池生成任务
- `schedule/init.go` - 定时任务调度器初始化

- `tools/ippool/` - 独立命令行工具
- `cmd/ippool_schedule/` - 定时任务调度器程序

## 输出文件

生成的 IP池文件：`ipPool_v1.json`

格式示例：
```json
{
  "192.168.31.254": "124.90.111.191",
  "172.18.0.1": "203.0.113.42"
}
```

## 编译

```bash
# 编译命令行工具
go build -o tools/ippool/ippool tools/ippool/main.go

# 编译定时任务调度器
go build -o cmd/ippool_schedule/ippool_schedule cmd/ippool_schedule/main.go
```

## 依赖

- `github.com/robfig/cron/v3` - 定时任务调度（已添加到 go.mod）
