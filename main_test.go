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

func TestKanbanWorkflow(t *testing.T) {
	a, tt := testApp(t)
	items := []tracker.Application{
		{Company: "远山科技", Role: "Go 后端", City: "上海", Source: "官网", Status: "已投递", Priority: "高", FollowUpDate: "2026-11-06", InterviewAt: "2026-11-07 14:30"},
		{Company: "星河实验室", Role: "数据工程师", City: "上海", Source: "其他", Status: "简历筛选", Priority: "普通"},
		{Company: "远山科技", Role: "平台工程师", City: "杭州", Source: "官网", Status: "简历筛选", Priority: "低"},
	}
	for i := range items {
		if err := a.store.Save(&items[i]); err != nil {
			t.Fatal(err)
		}
	}
	a.reload()
	tt.Frame()
	id := items[0].ID
	storedStatus := func() string {
		t.Helper()
		saved, err := a.store.List()
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range saved {
			if item.ID == id {
				return item.Status
			}
		}
		t.Fatal("card was removed from storage")
		return ""
	}
	if _, ok := tt.Find(fmt.Sprintf("编辑 %d", id)); !ok {
		t.Fatal("records must default to the list")
	}
	typeInto(t, tt, "筛选公司", "远山")
	typeInto(t, tt, "城市", "上海")
	click(t, tt, "看板")
	if !tt.HasText("1 条匹配 / 共 3 条投递") || !tt.HasText("已投递 · 1") || !tt.HasText("简历筛选 · 0") || tt.HasText("平台工程师") {
		t.Fatal("kanban did not reuse the list's combined filters")
	}
	for _, text := range []string{"Go 后端", "上海 · 官网", "优先级 · 高", "跟进 · 2026-11-06", "面试 · 2026-11-07 14:30", "放弃 · 0", "暂无投递"} {
		if !tt.HasText(text) {
			t.Fatalf("kanban missing %q", text)
		}
	}
	click(t, tt, fmt.Sprintf("修改状态 %d", id))
	click(t, tt, "简历筛选")
	if storedStatus() != "简历筛选" || a.page != "" || !tt.HasText("已投递 · 0") || !tt.HasText("简历筛选 · 1") {
		t.Fatal("changing a card status must persist and regroup, not open the editor")
	}
	click(t, tt, fmt.Sprintf("编辑投递 %d", id))
	typeInto(t, tt, "岗位 *", "Go 平台后端")
	click(t, tt, "确认保存")
	if !tt.HasText("Go 平台后端") || !tt.HasText("简历筛选 · 1") {
		t.Fatal("editing a card must return to the same filtered board")
	}
	click(t, tt, "状态")
	// Use the focused menu: a card's current status has the same visible text.
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	if a.filter.Status != "简历筛选" {
		t.Fatalf("status filter was not selected: %+v", a.filter)
	}
	click(t, tt, fmt.Sprintf("修改状态 %d", id))
	click(t, tt, "已投递")
	if storedStatus() != "已投递" || !tt.HasText("0 条匹配 / 共 3 条投递") || !tt.HasText("简历筛选 · 0") {
		t.Fatalf("status change under filter: stored=%q filter=%+v texts=%v", storedStatus(), a.filter, tt.Texts())
	}
	click(t, tt, "清空筛选")
	if !tt.HasText("已投递 · 1") || !tt.HasText("简历筛选 · 2") {
		t.Fatal("clear filters must restore all grouped cards")
	}
	click(t, tt, "列表")
	if !tt.HasText("3 条匹配 / 共 3 条投递") || !tt.HasText("Go 平台后端") {
		t.Fatal("switching back to the list lost the saved edit or records")
	}
}

func TestDemoWorkflow(t *testing.T) {
	a, tt := testApp(t)
	click(t, tt, "载入示例数据")
	if !tt.HasText("将添加 18 条示例数据，可随时在设置里清除") {
		t.Fatal("loading must explain the count and cleanup before confirmation")
	}
	click(t, tt, "取消")
	if len(a.items) != 0 {
		t.Fatal("canceling demo load added records")
	}
	click(t, tt, "+ 新增投递")
	typeInto(t, tt, "公司 *", "真实投递")
	typeInto(t, tt, "岗位 *", "用户自己的岗位")
	click(t, tt, "确认保存")
	realID := a.items[0].ID
	click(t, tt, "设置")
	click(t, tt, "载入示例数据")
	click(t, tt, "确认载入")
	if len(a.items) != 19 || !tt.HasText("重新载入") || !a.hasDemo() {
		t.Fatal("settings did not load demos alongside the real record")
	}
	click(t, tt, "投递记录")
	if !tt.HasText("示例") {
		t.Fatal("list must mark demo records")
	}
	click(t, tt, "看板")
	var demoID int64
	for _, item := range a.items {
		if item.IsDemo {
			demoID = item.ID
			break
		}
	}
	click(t, tt, fmt.Sprintf("编辑投递 %d", demoID))
	typeInto(t, tt, "岗位 *", "编辑过的示例岗位")
	click(t, tt, "确认保存")
	if !tt.HasText("编辑过的示例岗位") || !tt.HasText("示例") {
		t.Fatal("edited demo must remain visible and marked on the board")
	}
	click(t, tt, "设置")
	click(t, tt, "重新载入")
	click(t, tt, "取消")
	if len(a.items) != 19 {
		t.Fatal("canceling reload changed the dataset")
	}
	click(t, tt, "重新载入")
	click(t, tt, "确认载入")
	if len(a.items) != 19 {
		t.Fatal("reloading duplicated demo records")
	}
	click(t, tt, "清除示例数据")
	click(t, tt, "取消")
	if len(a.items) != 19 {
		t.Fatal("canceling cleanup removed records")
	}
	click(t, tt, "清除示例数据")
	click(t, tt, "确认清除")
	stored, err := a.store.List()
	if err != nil || len(stored) != 1 || stored[0].ID != realID || stored[0].IsDemo || stored[0].Role != "用户自己的岗位" {
		t.Fatalf("cleanup must preserve the real record: %+v, %v", stored, err)
	}
	if !tt.HasText("载入示例数据") || a.hasDemo() {
		t.Fatal("cleanup must reset the load action")
	}
	click(t, tt, "投递记录")
	if !tt.HasText("1 条匹配 / 共 1 条投递") || !tt.HasText("用户自己的岗位") {
		t.Fatal("board did not refresh after demo cleanup")
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
