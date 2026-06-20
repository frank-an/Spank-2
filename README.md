# Spank-2

[**English README**](./README_EN.md)

拍打 MacBook 笔记本时自动模拟按键或输入文字。

利用 Apple Silicon 加速计检测敲击/拍打，通过 CoreGraphics 模拟键盘事件。

## 要求

- **Apple Silicon Mac**（M1+）
- **sudo 权限**（加速计需要 IOKit 直接访问）
- **辅助功能权限**：系统设置 > 隐私与安全性 > 辅助功能 > 允许终端

# 项目结构

- **source-file**
  - **go.mod**: Go 模块定义
  - **go.sum**: 依赖校验
  - **main.go**: 主程序
  - **input_darwin.go**: macOS 底层输入模拟
- **README.MD**: 此文档

## 命令

| 命令 | 说明 |
|------|------|
| `sudo spank` | 默认模式，拍一下按回车 |
| `sudo spank --help` | 查看帮助 |

### --key / -k

指定按下的按键。

```bash
sudo spank --key space          # 按空格
sudo spank --key escape         # 按 Esc
sudo spank --key enter          # 按回车（默认）
sudo spank --key a              # 按 A
sudo spank --key tab            # 按 Tab
sudo spank --key backspace      # 按退格
sudo spank --key up             # 按上方向键
sudo spank --key down           # 按下方向键
sudo spank --key left           # 按左方向键
sudo spank --key right          # 按右方向键
sudo spank --key 0-9            # 按数字键
```

### --mouse / -m

模拟鼠标点击。

```bash
sudo spank --mouse 0            # 拍一下鼠标左键
sudo spank --mouse 1            # 拍一下鼠标右键
```

### --command / -c

拍打时执行终端命令。同时支持 `s=` 分段语法，每次拍打切换执行不同命令。

```bash
sudo spank -c 'open ~/.zshrc'                    # 拍一下打开 .zshrc
sudo spank -c 's=| open ~/.zshrc|quit TextEdit'  # 拍打循环：打开→退出
```

注意：命令以原用户身份执行，会加载 `~/.zshrc`，自定义 alias/函数可用。命令本身的输出不会显示。

### -v（灵敏度）

| 值 | 说明 |
|----|------|
| `high` | 敲桌子、碰一下都触发 |
| `mid` | 适合大多数场景（默认） |
| `low` | 得使劲敲才触发 |

```bash
sudo spank -v high              # 高灵敏度
sudo spank -v low --mouse 0     # 低灵敏度 + 鼠标左键
```

### --word（文本模式）

`--word` 模式改变 `--key` 的行为：

- **单个字符** → 直接模拟按键
- **多个字符** → 复制到剪贴板 + 模拟 `Cmd+V` 粘贴输入
- **分段输出** → 用 `s=` 前缀指定分隔符，每次拍打输出下一段，循环往复

```bash
sudo spank --word --key hello          # 拍一下输入 hello
sudo spank --word --key "你好世界"       # 拍一下输入 你好世界
sudo spank --word --key a               # 拍一下输入 a（单个字符=按键）
sudo spank --word --key 's=."Hi.I'm.Claude.'  # 分段输出 Hi → I'm → Claude
```

#### 转义序列

| 序列 | 效果 |
|------|------|
| `\n` | 换行 |
| `\t` | 制表符 |
| `\\` | 反斜杠本身 |

```bash
sudo spank --word --key "line1\nline2"    # 拍一下输入两行
sudo spank --word --key "a\tb\tc"         # 拍一下输入制表符分隔
```

#### 原始字符串

用 `r"..."` 包裹则不解析转义，按原样输入。

```bash
sudo spank --word --key 'r"hello\n"'      # 按原样输入 hello\n（不换行）
```

## 组合示例

### 日常使用

```bash
# 摸鱼快捷键：拍一下发消息
sudo spank -v high --word --key "/whisper\n"

# 高频场景：拍一下粘贴常用命令
sudo spank --word --key "sudo systemctl restart nginx\n"

# 拍一下输入邮箱
sudo spank --word --key "example@gmail.com"

# 拍一下带格式输入
sudo spank --word --key "姓名:\t张三\n年龄:\t25\n"

# 低灵敏度 + 方向键翻页
sudo spank -v low --key space

# 拍一下按 Tab 切换焦点
sudo spank --key tab
```

### 分段循环输出

```bash
# 聊天常用语循环
sudo spank -w --key 's=."好的 ."收到 ."马上来 .'

# 逐行输入命令
sudo spank -w --key 's=,git add .\n,git commit -m "update"\n,git push\n'

# 电话话术分段
sudo spank -w --key 's=|您好，请问有什么可以帮您？|感谢您的来电，再见！|'

# 分步操作引导
sudo spank -w --key 's=.第一步：打开设置.第二步：点击账号.第三步：退出登录.'
```

### 组合用法

```bash
# 高灵敏度 + 换行文本
sudo spank -v high --word --key "docker ps\n"

# 低灵敏度 + 分段输出（使劲敲才切换下一段）
sudo spank -v low -w --key 's=.nextpage.'

# 鼠标左键 + 低灵敏度
sudo spank -v low --mouse 0
```

### 实用场景

```bash
# 快速回复常用消息
sudo spank -w --k 's=,好的马上到！,收到，谢谢！,稍后回复你，,'

# 输入代码片段
sudo spank -w --k 's=.import os.\nimport sys.\nimport json'

# 粘贴 API 密钥
sudo spank --word --k "sk-xxxxxxxxxxxxxxxx"

# 多行地址录入
sudo spank --word --k "广东省深圳市\n南山区\n科技园南路\n"

# 配合转义的复杂文本
sudo spank -w --k "# 标题\n\n这是**粗体**文本\n- 列表项1\n- 列表项2\n"

# 逐条发送消息
sudo spank -w --k 's=|你好！|在吗？|有个事想请教一下|算了没事了|'
```

## 构建

```bash
cd source-file
GOFLAGS=-mod=mod go build -o ../spank -ldflags="-s -w" .
```

## 技术细节

- 通过 IOKit 直接读取 `AppleSPUHIDDevice` 加速计（`taigrr/apple-silicon-accelerometer` 库）
- 使用 CoreGraphics CGEvent 模拟键盘/鼠标事件
- 粘贴模式：`pbcopy` + `Cmd+V`（支持中文等任意 Unicode 字符）
- 零外部依赖，编译后单文件二进制

Changes were made based on the [Spank project](https://github.com/taigrr/spank).

