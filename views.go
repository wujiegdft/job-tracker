package main

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	"job-tracker-native/internal/tracker"
)

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Gap(0).Background(t.Background).Children(func() {
		ui.Column(c).Width(206).FillHeight().Padding(22, 16).Gap(12).
			Background(t.Surface).BorderWidth(0, 1, 0, 0).BorderColor(t.Border).Children(func() {
			ui.Text(c, "投递簿").FontSize(27).Bold().TextColor(t.Accent)
			ui.Text(c, "JOB TRACKER").FontSize(11).TextColor(t.TextMuted)
			ui.Divider(c).Margin(12, 0)
			ui.Button(c, "投递记录").OnClick(func() { a.navigate("") })
			ui.Button(c, "AI 智能录入").OnClick(func() { a.navigate("ai") })
			ui.Button(c, "设置").OnClick(func() { a.navigate("settings") })
			ui.Divider(c).Margin(12, 0)
			ui.Text(c, "按公司查看").FontSize(12).TextColor(t.TextMuted)
			ui.Button(c, "全部公司").OnClick(func() { a.filter.Company = ""; a.navigate("") })
			ui.Scroll(c.Key("companies")).Grow(1).Gap(6).Children(func() {
				for _, company := range a.companies() {
					ui.Button(c.Key(company), company).OnClick(func() { a.filter.Company = company; a.navigate("") })
				}
			})
			ui.Text(c, "本地优先 · 由你掌控").FontSize(11).TextColor(t.TextMuted)
		})
		ui.Column(c).Grow(1).MinWidth(0).FillHeight().Padding(28).Gap(18).Children(func() {
			ui.Column(c.Key(a.page)).Grow(1).Gap(18).Children(func() {
				switch a.page {
				case "edit":
					a.editor(c)
				case "ai":
					a.smartEntry(c)
				case "settings":
					a.settings(c)
				default:
					a.list(c)
				}
			})
			if a.message != "" {
				color := t.Accent
				if a.failed {
					color = t.Danger
				}
				ui.Text(c, a.message).TextColor(color).FontSize(13)
			}
		})
	})
	if ui.AlertDialog(c, &a.askDelete, "删除这条投递记录？", "记录将从本机永久删除，无法撤销。", "取消", "确认删除") == 1 {
		a.remove()
	}
	if a.askLoadDemo {
		text := fmt.Sprintf("将添加 %d 条示例数据，可随时在设置里清除。", len(tracker.DemoApplications(time.Now())))
		if a.hasDemo() {
			text += "现有示例将被替换（含编辑过的示例），真实记录不受影响。"
		}
		if ui.AlertDialog(c, &a.askLoadDemo, "载入示例数据？", text, "取消", "确认载入") == 1 {
			a.loadDemo()
		}
	}
	if ui.AlertDialog(c, &a.askClearDemo, "清除示例数据？", "仅删除带“示例”标记的记录（含编辑过的示例），保留全部真实记录。此操作无法撤销。", "取消", "确认清除") == 1 {
		a.clearDemo()
	}
}

func heading(c *ui.Context, title, subtitle string) {
	ui.Column(c).Grow(1).Gap(5).Children(func() {
		ui.Text(c, title).FontSize(26).Bold()
		ui.Text(c, subtitle).FontSize(13).TextColor(c.Theme().TextMuted)
	})
}

func inputField(c *ui.Context, label string, value *string, placeholder string) {
	ui.Field(c, label, func() {
		ui.TextInput(c, value).Label(label).Placeholder(placeholder)
	}).Grow(1)
}

func selectField(c *ui.Context, label string, value *string, options []string) {
	ui.Field(c, label, func() { ui.Select(c, value, options).Label(label) }).Grow(1)
}

func (a *app) list(c *ui.Context) {
	ui.Row(c).Gap(12).Children(func() {
		heading(c, "每一次投递，都有迹可循", "从公司出发，找到下一步机会。列表与看板共用筛选，状态可直接修改。")
		ui.PrimaryButton(c, "+ 新增投递").OnClick(func() { a.edit(tracker.NewApplication(), false) })
	})
	ui.Grid(c).Columns(3).AlignItems(ui.End).Padding(18).GapX(12).GapY(14).
		Radius(12).Background(c.Theme().Surface).Border(1, c.Theme().Border).Children(func() {
		ui.Field(c, "公司 · 优先检索", func() {
			ui.Autocomplete(c, &a.filter.Company, a.companies()).Label("筛选公司").Placeholder("输入公司名称…").Height(36)
		}).Grow(1)
		ui.Field(c, "岗位", func() {
			ui.TextInput(c, &a.filter.Role).Label("岗位").Placeholder("例如 Go 后端").Height(36)
		}).Grow(1)
		ui.Field(c, "城市", func() {
			ui.TextInput(c, &a.filter.City).Label("城市").Placeholder("例如 上海").Height(36)
		}).Grow(1)
		ui.Field(c, "来源", func() {
			ui.Select(c, &a.filter.Source, a.sources()).Label("来源").Height(36)
		}).Grow(1)
		ui.Field(c, "状态", func() {
			ui.Select(c, &a.filter.Status, append([]string{""}, tracker.Statuses...)).Label("状态").Height(36)
		}).Grow(1)
		ui.Button(c, "清空筛选").Height(36).FillWidth().OnClick(func() { a.filter = tracker.Filter{} })
	})
	visible := make([]*tracker.Application, 0, len(a.items))
	for i := range a.items {
		if tracker.Matches(a.items[i], a.filter) {
			visible = append(visible, &a.items[i])
		}
	}
	ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Gap(3).Children(func() {
			ui.Textf(c, "%d 条匹配 / 共 %d 条投递", len(visible), len(a.items)).FontSize(13).TextColor(c.Theme().TextMuted)
			if a.recordsView == 1 {
				ui.Text(c, "横向滚动查看全部状态 · 点击卡片正文编辑 · 下拉修改状态即保存").FontSize(11).TextColor(c.Theme().TextMuted)
			}
		})
		ui.Segmented(c, &a.recordsView, "列表", "看板").Label("投递视图")
	})
	if a.recordsView == 1 {
		a.kanban(c, visible)
		return
	}
	if len(visible) == 0 {
		ui.Column(c).Grow(1).Center().Gap(12).Children(func() {
			ui.Text(c, "下一份机会，从这里开始").FontSize(21).Bold()
			ui.Text(c, "暂无匹配记录。新增一条投递，或清空筛选条件。").TextColor(c.Theme().TextMuted)
			ui.Row(c).Gap(12).Children(func() {
				ui.Button(c, "从招聘文字或截图录入").OnClick(func() { a.navigate("ai") })
				a.demoLoadButton(c)
			})
		})
		return
	}
	cols := []ui.TableColumn{
		{Title: "公司", Width: 150}, {Title: "岗位", Width: 180}, {Title: "城市", Width: 76},
		{Title: "来源", Width: 90}, {Title: "状态 · 直接修改", Width: 130},
		{Title: "投递日期", Width: 105}, {Title: "操作", Width: 150},
	}
	a.table.Key = func(i int) any { return visible[i].ID }
	ui.Table(c, &a.table, cols, len(visible), func(row, col int) {
		item := visible[row]
		switch col {
		case 0:
			ui.Column(c).Gap(3).Padding(8, 0).Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, item.Company).Bold().SingleLine().Grow(1).MinWidth(0).Tooltip(item.Company)
					if item.IsDemo {
						ui.Text(c, "示例").FontSize(10).Padding(2, 5).Radius(4).Background(c.Theme().SurfaceHover).TextColor(c.Theme().Accent).Shrink(0)
					}
				})
				ui.Text(c, "优先级 · "+item.Priority).FontSize(11).TextColor(c.Theme().TextMuted)
			})
		case 1:
			ui.Text(c, item.Role).SingleLine()
		case 2:
			ui.Text(c, item.City).SingleLine()
		case 3:
			ui.Text(c, item.Source).SingleLine()
		case 4:
			ui.Select(c, &item.Status, tracker.Statuses).MinWidth(0).FillWidth().Label(fmt.Sprintf("修改状态 %d", item.ID)).OnChange(func() { a.saveStatus(item) })
		case 5:
			ui.Text(c, item.AppliedDate).FontSize(12)
		case 6:
			ui.Row(c).Gap(6).Children(func() {
				ui.Button(c, "编辑").Label(fmt.Sprintf("编辑 %d", item.ID)).OnClick(func() { a.edit(*item, false) })
				ui.Button(c, "删除").Label(fmt.Sprintf("删除 %d", item.ID)).OnClick(func() { a.deleteID, a.askDelete = item.ID, true })
			})
		}
	}).Grow(1).Border(1, c.Theme().Border).Radius(10)
}

func (a *app) kanban(c *ui.Context, visible []*tracker.Application) {
	t := c.Theme()
	groups := tracker.GroupByStatus(visible)
	ui.ScrollHorizontal(c.Key("kanban")).Grow(1).MinWidth(0).MinHeight(0).Gap(14).Padding(0, 0, 10, 0).AlignItems(ui.Stretch).Children(func() {
		for _, status := range tracker.Statuses {
			items := groups[status]
			ui.Column(c.Key(status)).Width(256).Shrink(0).FillHeight().MinHeight(0).Padding(12).Gap(12).
				Radius(12).Background(t.Surface).Border(1, t.Border).Children(func() {
				ui.Row(c).Height(22).Shrink(0).AlignItems(ui.Center).Children(func() {
					ui.Textf(c, "%s · %d", status, len(items)).Bold().FontSize(14)
				})
				ui.Scroll(c.Key("cards")).Grow(1).MinHeight(0).Gap(12).Children(func() {
					if len(items) == 0 {
						ui.Text(c, "暂无投递").FontSize(13).TextColor(t.TextMuted).Padding(18, 0)
					}
					for _, item := range items {
						a.kanbanCard(c, item)
					}
				})
			})
		}
	})
}

func (a *app) kanbanCard(c *ui.Context, item *tracker.Application) {
	t := c.Theme()
	ui.Column(c.Key(item.ID)).FillWidth().Height(232).Shrink(0).Padding(12).Gap(10).
		Radius(10).Background(t.Background).Border(1, t.Border).Children(func() {
		body := ui.ButtonBase(c).Column().AlignItems(ui.Stretch).Justify(ui.Start).Grow(1).MinHeight(0).MinWidth(0).Gap(6).Radius(6).
			Label(fmt.Sprintf("编辑投递 %d", item.ID)).OnClick(func() { a.edit(*item, false) })
		if body.Hovered() {
			body.Background(t.SurfaceHover)
		}
		body.Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, item.Company).Bold().FontSize(16).SingleLine().Grow(1).MinWidth(0).Tooltip(item.Company)
				if item.IsDemo {
					ui.Text(c, "示例").FontSize(10).Padding(2, 5).Radius(4).Background(t.SurfaceHover).TextColor(t.Accent).Shrink(0)
				}
			})
			ui.Text(c, item.Role).FontSize(14).MaxLines(2).Height(40).Tooltip(item.Role)
			meta := item.Source
			if item.City != "" {
				meta = item.City + " · " + meta
			}
			ui.Text(c, meta).FontSize(12).TextColor(t.TextMuted).SingleLine().Tooltip(meta)
			color := t.TextMuted
			if item.Priority == "高" {
				color = t.Accent
			}
			ui.Textf(c, "优先级 · %s", item.Priority).FontSize(12).TextColor(color)
			ui.Column(c).Height(38).Shrink(0).Gap(2).Children(func() {
				if item.FollowUpDate != "" {
					ui.Text(c, "跟进 · "+item.FollowUpDate).FontSize(12).SingleLine()
				}
				if item.InterviewAt != "" {
					ui.Text(c, "面试 · "+item.InterviewAt).FontSize(12).SingleLine()
				}
			})
		})
		// Commit after selection, before changing the card's group or filtering it out.
		status := item.Status
		ui.Select(c, &status, tracker.Statuses).MinWidth(0).FillWidth().Height(32).
			Label(fmt.Sprintf("修改状态 %d", item.ID)).OnChange(func() { item.Status = status; a.saveStatus(item) })
	})
}

func (a *app) editor(c *ui.Context) {
	title := "新增投递"
	if a.draft.ID != 0 {
		title = "编辑投递"
	} else if a.draftAI {
		title = "确认 AI 草稿"
	}
	ui.Row(c).Gap(12).Children(func() {
		heading(c, title, "公司与岗位必填；日期 YYYY-MM-DD，面试时间 YYYY-MM-DD HH:MM（本地时间）。")
		ui.Button(c, "取消").OnClick(func() { a.navigate("") })
		ui.PrimaryButton(c, "确认保存").OnClick(a.save)
	})
	ui.Scroll(c.Key(a.draft.ID)).Grow(1).Gap(18).Padding(20).Radius(12).Border(1, c.Theme().Border).Children(func() {
		d := &a.draft
		ui.Text(c, "01 / 机会信息").Bold().TextColor(c.Theme().Accent)
		ui.Row(c).Gap(16).Children(func() {
			inputField(c, "公司 *", &d.Company, "公司全称或常用名")
			inputField(c, "岗位 *", &d.Role, "岗位名称")
		})
		ui.Row(c).Gap(16).Children(func() {
			inputField(c, "城市", &d.City, "上海 / 远程")
			inputField(c, "薪资（纯文本）", &d.Salary, "例如 25–35K · 16 薪")
		})
		ui.Row(c).Gap(16).Children(func() {
			ui.Field(c, "来源（可自定义）", func() {
				ui.Autocomplete(c, &d.Source, tracker.Sources).Label("录入来源").Placeholder("选择建议或直接输入")
			}).Grow(1)
			label := "来源人（内推 / 猎头必填）"
			if d.Source == "内推" {
				label = "内推人 *"
			} else if d.Source == "猎头" {
				label = "猎头姓名 *"
			}
			inputField(c, label, &d.SourcePerson, "姓名或称呼")
		})
		inputField(c, "岗位链接", &d.URL, "https://…")
		ui.Divider(c)
		ui.Text(c, "02 / 投递进度").Bold().TextColor(c.Theme().Accent)
		ui.Row(c).Gap(16).Children(func() {
			selectField(c, "当前状态", &d.Status, tracker.Statuses)
			selectField(c, "优先级", &d.Priority, tracker.Priorities)
		})
		ui.Row(c).Gap(16).Children(func() {
			inputField(c, "投递日期", &d.AppliedDate, "YYYY-MM-DD")
			inputField(c, "下次跟进日期", &d.FollowUpDate, "YYYY-MM-DD（可空）")
		})
		ui.Row(c).Gap(16).Children(func() {
			inputField(c, "面试时间", &d.InterviewAt, "YYYY-MM-DD HH:MM（可空）")
			inputField(c, "联系人", &d.Contact, "姓名、电话或邮箱")
		})
		ui.Divider(c)
		ui.Text(c, "03 / 原文与备忘").Bold().TextColor(c.Theme().Accent)
		ui.Field(c, "JD 原文", func() { ui.TextArea(c, &d.JD).Label("JD 原文").Height(180) })
		ui.Field(c, "备注", func() { ui.TextArea(c, &d.Notes).Label("备注").Height(110) })
	})
}

func (a *app) smartEntry(c *ui.Context) {
	ui.Row(c).Children(func() {
		heading(c, "把招聘信息，变成一条机会", "文字与截图可同时提交。AI 提取后进入确认页，不会自动保存。")
	})
	ui.Scroll(c).Grow(1).Gap(18).Children(func() {
		ui.Column(c).Padding(20).Gap(14).Radius(12).Border(1, c.Theme().Border).Children(func() {
			ui.Text(c, "招聘文字 / JD").Bold()
			ui.TextArea(c, &a.input).Label("招聘文字").Placeholder("粘贴职位描述、聊天记录或招聘信息…").Height(240).Disabled(a.busy).
				OnPaste(func(text string) bool {
					if text == "" && !a.busy {
						a.pasteImage()
						return true
					}
					return false
				})
			ui.Row(c).Gap(10).Children(func() {
				ui.Button(c, "选择图片").Disabled(a.busy || a.loading).OnClick(a.chooseImage)
				ui.Button(c, "粘贴截图").Disabled(a.busy || a.loading).OnClick(a.pasteImage)
				ui.Text(c, "PNG / JPEG / WebP · 最大 10 MiB").FontSize(12).TextColor(c.Theme().TextMuted)
			})
			if a.preview != nil {
				ui.Row(c).Gap(14).Children(func() {
					ui.Image(c, a.preview).Size(96, 96).Fit(ui.Contain).Radius(6)
					ui.Text(c, a.imageName).Grow(1).SingleLine()
					ui.Button(c, "移除图片").Disabled(a.busy).OnClick(func() { a.image, a.preview, a.imageName = nil, nil, "" })
				})
			}
		})
		ui.Text(c, "隐私提示：仅点击生成时，将上述文字和图片发送至 DeepSeek。请先遮挡个人敏感信息。").FontSize(13).TextColor(c.Theme().TextMuted)
		ui.Row(c).Gap(12).Children(func() {
			ui.PrimaryButton(c, "生成待确认草稿").Disabled(a.busy || a.loading).OnClick(a.extract)
			if a.busy {
				ui.Text(c, "正在提取…").TextColor(c.Theme().Accent)
				ui.Button(c, "取消请求").OnClick(a.cancelExtraction)
			}
		})
	})
}

func (a *app) settings(c *ui.Context) {
	ui.Row(c).Children(func() {
		heading(c, "DeepSeek 设置", "只需填写 API key；接口地址与模型固定，无需选择。")
	})
	ui.Scroll(c).Grow(1).Gap(18).Children(func() {
		ui.Column(c).MaxWidth(750).Padding(22).Gap(20).Radius(12).Border(1, c.Theme().Border).Children(func() {
			ui.Field(c, "DeepSeek API key", func() { ui.TextInput(c, &a.config.APIKey).Label("DeepSeek API key").Password().Placeholder("sk-…") })
			ui.Text(c, "固定接口："+tracker.DeepSeekBaseURL+" · 模型："+tracker.DeepSeekModel).FontSize(13).TextColor(c.Theme().TextMuted)
			ui.Text(c, "API key 以明文保存在本机 config.json 中（Unix 权限 0600），未使用系统钥匙串；请勿共享此文件。").FontSize(13).TextColor(c.Theme().TextMuted)
			ui.Text(c, a.configPath).FontSize(12).TextColor(c.Theme().TextMuted)
			ui.PrimaryButton(c, "保存设置").OnClick(a.saveConfig)
		})
		ui.Text(c, "无需配置 AI，也可使用全部手动录入、检索和状态管理功能。").TextColor(c.Theme().TextMuted)
		ui.Column(c).MaxWidth(750).Padding(22).Gap(16).Radius(12).Border(1, c.Theme().Border).Children(func() {
			ui.Text(c, "演示数据").FontSize(20).Bold()
			ui.Text(c, "18 条虚构投递，涵盖全部状态与来源；跟进和面试日期按载入当天生成。").TextColor(c.Theme().TextMuted)
			ui.Text(c, "带“示例”标记的记录在编辑后仍是示例；重新载入会替换旧示例，清除不会影响真实记录。").FontSize(13).TextColor(c.Theme().TextMuted)
			ui.Row(c).Gap(12).Children(func() {
				a.demoLoadButton(c)
				ui.Button(c, "清除示例数据").Disabled(!a.hasDemo()).OnClick(func() { a.askClearDemo = true })
			})
		})
	})
}

func (a *app) demoLoadButton(c *ui.Context) {
	label := "载入示例数据"
	if a.hasDemo() {
		label = "重新载入"
	}
	ui.Button(c, label).OnClick(func() { a.askLoadDemo = true })
}
