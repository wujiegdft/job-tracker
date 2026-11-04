# 投递簿 · Job Tracker

一个本地优先的桌面求职投递追踪器：按公司梳理机会，直接更新投递状态，把招聘文字或截图转成待确认草稿。

使用 [MyGo](https://github.com/egoist/mygo) **原生 UI（`ui` 包）**，全部界面由 Go 编写。无 HTML、JavaScript、Bun、WebView；SQLite 使用纯 Go 驱动 [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite)，无需 cgo 或 C 编译器。

![投递列表：公司优先检索与行内状态修改](docs/screenshots/list.png)

> 截图均为真实 Linux 原生窗口，使用虚构演示数据；AI 截图使用本地模拟接口，不代表真实模型的识别准确率。首次启动数据库为空，不会自动导入示例数据。

## 快速运行

需要 **Go 1.27.3+**，以及桌面图形环境。进入项目根目录后：

```sh
go run .
```

开发时自动重建：

```sh
go tool mygo dev
```

平台要求：

| 平台 | 运行时依赖 |
| --- | --- |
| macOS 12+ | 无额外依赖 |
| Windows 10 / 11 | 无额外依赖，无需 WebView2 |
| Linux x64 / arm64 | GTK 3、OpenGL；建议安装中文字体，无需 WebKitGTK |

Debian / Ubuntu：

```sh
sudo apt-get install libgtk-3-0 libgl1 fonts-noto-cjk
go run .
```

项目由官方模板初始化：

```sh
go run github.com/egoist/mygo/cmd/mygo@latest init -template native job-tracker
```

## 使用

### 手动投递与检索

1. 点击「新增投递」，填写公司、岗位，确认保存。**内推 / 猎头来源还必须填写对应姓名**。
2. 公司、岗位、城市支持不区分大小写的子串匹配；来源、状态为精确匹配。五个条件取交集（AND），空值表示不限。
3. 左侧公司列表可快速填入公司条件；「全部公司」只清除公司条件，「清空筛选」清除全部条件。
4. 直接使用列表行内的状态下拉框，无需打开编辑页；更新后立即写入 SQLite。
5. 点击「编辑」查看或修改全部字段；长表单可滚动，保存按钮固定在上方。「取消」不保存当前表单；切换其他页面同样不保存。
6. 删除前有二次确认，确认后不可撤销。

列表按投递日期从新到旧排序，同一天按新增顺序倒序。窄窗口可横向滚动表格查看操作列；长公司名和岗位名可在编辑页完整查看。

**记录字段：**

- 机会：公司、岗位、城市、来源、内推人 / 猎头姓名、岗位链接、薪资。
- 进度：状态、投递日期、下次跟进日期、面试时间、优先级。
- 详情：JD 原文、联系人、备注。

来源提供内推、Boss直聘、猎聘、猎头、官网、其他建议，也可直接输入自定义值；保存过的来源会出现在筛选选项中。

状态：已投递、简历筛选、笔试、一面、二面、HR面、Offer、被拒、放弃。优先级：普通、高、低。薪资为可选纯文本，不参与筛选。

日期格式为 `YYYY-MM-DD`，面试时间为 `YYYY-MM-DD HH:MM`，按设备本地时区填写；可留空。新记录默认当天、已投递、普通优先级。**记录跟进日期不等于发送提醒**。

### AI 智能录入

1. 在「设置」中填写兼容 OpenAI 的接口地址（Base URL）、模型名与 API key，点击「保存设置」。无需配置 AI 也可使用全部手动功能。
2. 打开「AI 智能录入」，粘贴招聘文字 / JD；也可「粘贴截图」，或通过「选择图片」加载文件。文字与图片可同时提交。
3. 点击「生成待确认草稿」，提取在后台执行；可取消请求，切换页面也会取消，过期结果不会覆盖新页面。
4. 在「确认 AI 草稿」页逐项修正，点击「确认保存」才写入数据库。「取消」不会保存。

![文字与图片同时录入](docs/screenshots/ai-entry.png)

![人工确认 AI 草稿](docs/screenshots/ai-draft.png)

接口规则与限制：

- Base URL 可填 `https://api.openai.com/v1`，也可填完整的 `/chat/completions` 地址；支持 query 参数。
- 默认模型名为 `gpt-4o-mini`；截图需要服务端模型支持视觉输入。
- 使用 `POST /chat/completions`、Bearer token、`response_format: {"type":"json_object"}`，图片通过 base64 data URL 发送。服务端需兼容这些字段；不支持只提供 Responses API 的服务。
- PNG / JPEG / WebP，最大 **10 MiB**。剪贴板无图片或平台读取失败时，改用文件选择；当前未实现拖放。
- 请求超时 **45 秒**，响应上限 **2 MiB**。遇到 HTTP 错误、格式错误或缺失配置会显示提示，不会生成已保存记录。
- 仅提取公司、岗位、城市、来源、来源人、岗位链接、JD、联系人、备注、薪资；未知信息留空。模型不能设置记录 ID、状态、日期或优先级；这些采用本地默认值。
- 模型未返回 JD 时，保留粘贴文字。只有截图时没有本地 OCR，依赖视觉模型。
- 模型可能遗漏或误判；**必须人工核对**，草稿缺少必填字段时不能保存。

### 隐私与本地文件

记录保存在本机 SQLite，无云同步。仅点击 AI 生成按钮时，将录入页的文字和图片发送给设置的服务；不会附带其他投递记录。没有后台遥测。

默认目录由 Go 的 `os.UserConfigDir()` 决定：

| 平台 | 默认位置 |
| --- | --- |
| Linux | `$XDG_CONFIG_HOME/job-tracker`，未设置时 `~/.config/job-tracker` |
| macOS | `~/Library/Application Support/job-tracker` |
| Windows | `%AppData%\job-tracker` |

目录内：`applications.db` 保存投递；`config.json` 保存 AI 设置。新建目录权限为 `0700`，数据库与配置文件在 Unix 上为 `0600`；Windows 权限由系统 ACL 决定。

**API key 为本地明文，不使用系统钥匙串。** 设置页输入框隐藏 key；不要分享配置文件或把真实数据提交到 GitHub。日志与接口错误不会输出 key 或服务端原始响应。HTTP 接口不加密传输，请仅对可信本地服务使用 HTTP，远程接口使用 HTTPS。

可指定独立目录进行演示、测试或便携使用：

```sh
go run . --data-dir ./my-private-data
```

备份时先退出应用，再复制整个数据目录；恢复时同样先退出。不要同时运行多个实例编辑同一数据库；MVP 没有跨实例实时刷新。

## 构建与验证

```sh
go test ./...
go vet ./...
go tool mygo vet .
go tool mygo build
```

构建输出在 `build/<os>-<arch>/`，包含原生可执行文件和平台安装包。也可跨平台打包（签名、公证以及 macOS DMG 制作需要对应环境）：

```sh
go tool mygo build -platform darwin/universal,windows/amd64,linux/amd64
```

验证记录：

- `CGO_ENABLED=0 go test -count=1 ./...`：通过；覆盖 SQLite 全字段重开持久化、CRUD、日期校验、五维筛选、配置权限，以及 AI 多模态请求、取消与错误处理。
- `go test -race -count=1 ./...`、`go vet ./...`、`go tool mygo vet .`：通过。
- MyGo 无窗口交互测试（`ui.NewTester`）：新增 / 编辑 / 删除确认、筛选、筛选中行内改状态、设置保存、AI 草稿确认前不落库。
- Linux 真实原生窗口：通过文件选择加载招聘截图，文字 + 图片发送到本地模拟接口，收到草稿后确认写入 SQLite；鼠标操作状态下拉、编辑与删除取消，检查数据库结果。已检查实际渲染截图及交互录屏。
- `go tool mygo build -platform linux/amd64`：成功生成可执行文件、`.deb` 和 `.tar.gz`。Windows amd64 / macOS arm64 仅验证交叉编译，未进行真实系统运行测试。
- 未接入真实付费模型；不同厂商的兼容性与识别准确率需使用自己的服务验证。Linux orb 无 GPU，窗口使用 MyGo 的 CPU 绘制回退。

## 代码组织

- `main.go`：数据目录、应用生命周期与原生窗口。
- `app.go`：界面状态、持久化动作、图片附件与可取消的后台提取。
- `views.go`：列表、筛选、编辑、AI 录入、设置；无 webview。
- `internal/tracker/`：领域校验、SQLite、配置文件、OpenAI 兼容客户端；不依赖 UI，可独立测试。
- `main_test.go`：从用户操作入口验证完整流程。

## Roadmap（尚未实现）

- [ ] 看板视图
- [ ] 跟进提醒
- [ ] CSV 导出
- [ ] 统计图
