package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"job-tracker-native/internal/tracker"
)

// All UI state belongs to the main thread. Workers publish via Window.Update.
type app struct {
	store      *tracker.Store
	win        *mygo.Window
	config     tracker.Config
	configPath string
	items      []tracker.Application
	filter     tracker.Filter
	table      ui.ListState
	page       string
	draft      tracker.Application
	draftAI    bool
	deleteID   int64
	askDelete  bool
	message    string
	failed     bool
	input      string
	image      []byte
	imageName  string
	preview    *ui.Bitmap
	loading    bool
	busy       bool
	cancel     context.CancelFunc
	requestID  int
}

func (a *app) fail(err error) {
	if err != nil {
		a.message, a.failed = err.Error(), true
	}
}

func (a *app) notify(message string) { a.message, a.failed = message, false }

func (a *app) reload() {
	items, err := a.store.List()
	if err != nil {
		a.fail(err)
		return
	}
	a.items = items
}

func (a *app) navigate(page string) {
	if a.page == "ai" && page != "ai" {
		a.cancelExtraction()
	}
	a.page, a.message = page, ""
}

func (a *app) edit(item tracker.Application, fromAI bool) {
	a.draft, a.draftAI = item, fromAI
	a.navigate("edit")
}

func (a *app) save() {
	if err := a.store.Save(&a.draft); err != nil {
		a.fail(err)
		return
	}
	a.page = ""
	a.reload()
	a.notify("已保存到本机")
}

func (a *app) saveStatus(item *tracker.Application) {
	if err := a.store.SetStatus(item.ID, item.Status); err != nil {
		a.fail(err)
		a.reload()
		return
	}
	a.notify("状态已更新")
}

func (a *app) remove() {
	if err := a.store.Delete(a.deleteID); err != nil {
		a.fail(err)
		return
	}
	a.reload()
	a.notify("投递记录已删除")
}

func (a *app) saveConfig() {
	if err := tracker.SaveConfig(a.configPath, a.config); err != nil {
		a.fail(err)
		return
	}
	a.notify("设置已保存")
}

func (a *app) sources() []string {
	values := append([]string{""}, tracker.Sources...)
	for _, item := range a.items {
		found := false
		for _, value := range values {
			found = found || value == item.Source
		}
		if !found {
			values = append(values, item.Source)
		}
	}
	return values
}

func (a *app) companies() []string {
	set := map[string]bool{}
	for _, item := range a.items {
		set[item.Company] = true
	}
	values := make([]string, 0, len(set))
	for value := range set {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func (a *app) attach(data []byte, name string) {
	if len(data) > 10<<20 {
		a.fail(fmt.Errorf("图片不能超过 10 MiB"))
		return
	}
	mime := http.DetectContentType(data)
	if mime != "image/png" && mime != "image/jpeg" && mime != "image/webp" {
		a.fail(fmt.Errorf("请选择 PNG、JPEG 或 WebP 图片"))
		return
	}
	preview, err := ui.DecodeBitmap(data)
	if err != nil {
		a.fail(fmt.Errorf("无法读取图片"))
		return
	}
	a.image, a.imageName, a.preview = data, name, preview
	a.notify("图片已附加；点击生成草稿后才会发送")
}

func (a *app) chooseImage() {
	a.loading = true
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent: a.win, Title: "选择招聘截图",
			Filters: []mygo.FileFilter{{Name: "招聘截图", Extensions: []string{"png", "jpg", "jpeg", "webp"}}},
		})
		var data []byte
		var name string
		if err == nil && len(paths) > 0 {
			name = filepath.Base(paths[0])
			var f *os.File
			f, err = os.Open(paths[0])
			if err == nil {
				data, err = io.ReadAll(io.LimitReader(f, (10<<20)+1))
				f.Close()
			}
		}
		a.win.Update(func() {
			a.loading = false
			if err != nil {
				a.fail(fmt.Errorf("读取图片失败，请重试"))
			} else if data != nil {
				a.attach(data, name)
			}
		})
	}()
}

func (a *app) pasteImage() {
	if data := mygo.Clipboard.ReadImage(); len(data) > 0 {
		a.attach(data, "剪贴板截图.png")
	} else {
		a.fail(fmt.Errorf("剪贴板中没有图片，请使用“选择图片”"))
	}
}

func (a *app) cancelExtraction() {
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	a.requestID++
	a.busy = false
}

func (a *app) extract() {
	if a.busy {
		return
	}
	if strings.TrimSpace(a.config.APIKey) == "" || strings.TrimSpace(a.config.Model) == "" {
		a.fail(fmt.Errorf("请先在设置中填写 API key 和模型名"))
		return
	}
	if strings.TrimSpace(a.input) == "" && len(a.image) == 0 {
		a.fail(fmt.Errorf("请先粘贴招聘文字或添加图片"))
		return
	}
	a.cancelExtraction()
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel, a.busy, a.message = cancel, true, ""
	id, config, input, image := a.requestID, a.config, a.input, a.image
	go func() {
		draft, err := tracker.Extract(ctx, config, input, image)
		cancel()
		a.win.Update(func() {
			if a.requestID != id {
				return // A canceled request must never replace a newer draft.
			}
			a.busy, a.cancel = false, nil
			if err != nil {
				a.fail(err)
				return
			}
			a.edit(draft, true)
			a.notify("AI 草稿未保存，请逐项核对后确认")
		})
	}()
}
