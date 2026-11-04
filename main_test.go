package main

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"job-tracker-native/internal/tracker"
)

func testApp(t *testing.T) (*app, *ui.Tester) {
	t.Helper()
	dir := t.TempDir()
	store, err := tracker.Open(filepath.Join(dir, "applications.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	a := &app{store: store, configPath: filepath.Join(dir, "config.json")}
	a.reload()
	return a, ui.NewTester(a.view, 1280, 860)
}

func click(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
}

func typeInto(t *testing.T, tt *ui.Tester, label, value string) {
	t.Helper()
	click(t, tt, label)
	tt.Key(ui.Ctrl, ui.KeyA)
	tt.Type(value)
}

func TestApplicationWorkflow(t *testing.T) {
	a, tt := testApp(t)
	click(t, tt, "+ 新增投递")
	click(t, tt, "确认保存")
	if !tt.HasText("公司") || a.page != "edit" {
		t.Fatal("invalid draft should remain open")
	}
	typeInto(t, tt, "公司 *", "远山科技")
	typeInto(t, tt, "岗位 *", "Go 平台工程师")
	typeInto(t, tt, "城市", "上海")
	click(t, tt, "确认保存")
	items, err := a.store.List()
	if err != nil || len(items) != 1 || items[0].Role != "Go 平台工程师" {
		t.Fatalf("saved records: %+v, %v", items, err)
	}
	id := items[0].ID
	click(t, tt, fmt.Sprintf("修改状态 %d", id))
	click(t, tt, "一面")
	items, _ = a.store.List()
	if items[0].Status != "一面" {
		t.Fatalf("inline status not persisted: %+v", items[0])
	}
	// Changing a filtered row's status must still persist when it leaves the list.
	click(t, tt, "状态")
	click(t, tt, "一面")
	click(t, tt, fmt.Sprintf("修改状态 %d", id))
	click(t, tt, "二面")
	items, _ = a.store.List()
	if items[0].Status != "二面" || !tt.HasText("0 条匹配 / 共 1 条投递") {
		t.Fatal("status change from a filtered row was lost")
	}
	click(t, tt, "清空筛选")
	click(t, tt, fmt.Sprintf("编辑 %d", id))
	typeInto(t, tt, "岗位 *", "Go 后端工程师")
	click(t, tt, "确认保存")
	typeInto(t, tt, "筛选公司", "不存在的公司")
	if !tt.HasText("0 条匹配 / 共 1 条投递") {
		t.Fatalf("company filter failed: %v", tt.Texts())
	}
	click(t, tt, "清空筛选")
	if !tt.HasText("Go 后端工程师") {
		t.Fatal("edited role missing after clearing filters")
	}
	click(t, tt, fmt.Sprintf("删除 %d", id))
	click(t, tt, "取消")
	items, _ = a.store.List()
	if len(items) != 1 {
		t.Fatal("canceling deletion removed the record")
	}
	click(t, tt, fmt.Sprintf("删除 %d", id))
	click(t, tt, "确认删除")
	items, _ = a.store.List()
	if len(items) != 0 {
		t.Fatal("confirmed deletion did not persist")
	}
}

func TestDraftAndSettings(t *testing.T) {
	a, tt := testApp(t)
	click(t, tt, "AI 智能录入")
	click(t, tt, "生成待确认草稿")
	if !tt.HasText("API key") {
		t.Fatal("missing configuration was not explained")
	}
	click(t, tt, "设置")
	if tt.HasText("接口地址（Base URL）") || tt.HasText("模型名") || !tt.HasText("DeepSeek 设置") {
		t.Fatal("settings should only expose a DeepSeek API key")
	}
	typeInto(t, tt, "DeepSeek API key", "test-only-key")
	click(t, tt, "保存设置")
	config, err := tracker.LoadConfig(a.configPath)
	if err != nil || config.APIKey != "test-only-key" {
		t.Fatalf("settings not persisted: %+v, %v", config, err)
	}
	// The confirmation screen uses the same public save path as manual entry.
	draft := tracker.NewApplication()
	draft.Company, draft.Role = "示例公司", "数据工程师"
	a.edit(draft, true)
	tt.Frame()
	if !tt.HasText("确认 AI 草稿") {
		t.Fatal("AI draft is not identified")
	}
	items, _ := a.store.List()
	if len(items) != 0 {
		t.Fatal("draft was saved before confirmation")
	}
	click(t, tt, "取消")
	items, _ = a.store.List()
	if len(items) != 0 {
		t.Fatal("canceling a draft saved it")
	}
	a.edit(draft, true)
	tt.Frame()
	click(t, tt, "确认保存")
	items, _ = a.store.List()
	if len(items) != 1 || items[0].Company != "示例公司" {
		t.Fatal("confirmed draft not saved")
	}
}
