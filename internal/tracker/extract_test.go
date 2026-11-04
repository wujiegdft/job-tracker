package tracker

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func writeCompletion(w http.ResponseWriter, answer string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"choices": []any{map[string]any{"message": map[string]string{"content": answer}}},
	})
}

func TestExtractMultimodalAndSafeDraft(t *testing.T) {
	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	text := "  招聘原文\n第二行  "
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer fake-key" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("incorrect endpoint, method or headers")
		}
		var request struct {
			Model          string `json:"model"`
			ResponseFormat struct {
				Type string `json:"type"`
			} `json:"response_format"`
			Messages []struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error("invalid request JSON")
			w.WriteHeader(400)
			return
		}
		if request.Model != "vision-model" || request.ResponseFormat.Type != "json_object" || len(request.Messages) != 2 {
			t.Error("missing model, structured format, or messages")
			w.WriteHeader(400)
			return
		}
		if request.Messages[0].Role != "system" || request.Messages[1].Role != "user" || !bytes.Contains(request.Messages[0].Content, []byte("不虚构")) {
			t.Error("missing extraction-only system instructions")
		}
		var parts []struct {
			Type     string `json:"type"`
			Text     string `json:"text"`
			ImageURL struct {
				URL string `json:"url"`
			} `json:"image_url"`
		}
		if err := json.Unmarshal(request.Messages[1].Content, &parts); err != nil || len(parts) != 2 {
			t.Error("invalid multimodal parts")
			w.WriteHeader(400)
			return
		}
		if parts[0].Type != "text" || parts[0].Text != text || parts[1].Type != "image_url" || parts[1].ImageURL.URL != "data:image/png;base64,"+base64.StdEncoding.EncodeToString(pngBytes.Bytes()) {
			t.Error("multimodal data did not preserve text and base64 image")
		}
		writeCompletion(w, `{"company":" 公司 ","role":"Go工程师","city":"杭州","source":"内推","source_person":"张三","url":"https://example.com/job","jd":"","contact":"招聘部","notes":"需核实","salary":"25-35K","id":{"malicious":true},"status":"Offer","priority":"高","applied_date":"1900-01-01","follow_up_date":"bad","interview_at":"2000-01-01 00:00"}`)
	}))
	defer srv.Close()
	before := time.Now().Format(time.DateOnly)
	a, err := Extract(context.Background(), Config{BaseURL: srv.URL + "/v1", Model: "vision-model", APIKey: "fake-key"}, text, pngBytes.Bytes())
	after := time.Now().Format(time.DateOnly)
	if err != nil {
		t.Fatal(err)
	}
	if a.Company != "公司" || a.Role != "Go工程师" || a.City != "杭州" || a.Source != "内推" || a.SourcePerson != "张三" || a.Contact != "招聘部" || a.Notes != "需核实" || a.Salary != "25-35K" || a.URL != "https://example.com/job" || a.JD != text {
		t.Fatalf("facts/fallback lost: %+v", a)
	}
	if a.ID != 0 || a.Status != "已投递" || a.Priority != "普通" || (a.AppliedDate != before && a.AppliedDate != after) || a.FollowUpDate != "" || a.InterviewAt != "" {
		t.Fatalf("model controlled protected fields: %+v", a)
	}
}

func TestExtractEndpointVariantsAndIncompleteDraft(t *testing.T) {
	for _, suffix := range []string{"/v1", "/v1/", "/v1/chat/completions", "/v1/chat/completions/", "/v1?api-version=test"} {
		t.Run(suffix, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/chat/completions" || (strings.Contains(suffix, "?") && r.URL.Query().Get("api-version") != "test") {
					t.Error("incorrect resolved endpoint")
				}
				var request struct {
					Messages []struct {
						Content json.RawMessage `json:"content"`
					} `json:"messages"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil || len(request.Messages) != 2 || string(request.Messages[1].Content) != `"only known text"` {
					t.Error("incorrect text-only message")
				}
				writeCompletion(w, `{"url":"javascript:alert(1)","jd":"extracted JD"}`)
			}))
			defer srv.Close()
			a, err := Extract(context.Background(), Config{srv.URL + suffix, "model", "fake-key"}, "only known text", nil)
			if err != nil || a.Company != "" || a.Role != "" || a.Source != "其他" || a.URL != "" || a.JD != "extracted JD" {
				t.Fatalf("incomplete draft rejected or unsafe URL accepted: %+v, %v", a, err)
			}
		})
	}
}

func TestExtractImageTypesAndInputValidation(t *testing.T) {
	var jpegBytes bytes.Buffer
	if err := jpeg.Encode(&jpegBytes, image.NewRGBA(image.Rect(0, 0, 1, 1)), nil); err != nil {
		t.Fatal(err)
	}
	webpBytes, err := base64.StdEncoding.DecodeString("UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA")
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		if !bytes.Contains(body, []byte("data:image/jpeg;base64,")) && !bytes.Contains(body, []byte("data:image/webp;base64,")) {
			t.Error("incorrect image-only MIME/data URL")
		}
		writeCompletion(w, `{}`)
	}))
	defer srv.Close()
	c := Config{srv.URL + "/v1", "model", "fake-key"}
	for _, imageBytes := range [][]byte{jpegBytes.Bytes(), webpBytes} {
		if _, err := Extract(context.Background(), c, "", imageBytes); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		config Config
		text   string
		image  []byte
	}{
		{c, " \t ", nil},
		{c, "text", []byte("not an image")},
		{c, "text", make([]byte, maxImageBytes+1)},
		{Config{c.BaseURL, "", c.APIKey}, "text", nil},
		{Config{c.BaseURL, c.Model, ""}, "text", nil},
		{Config{"file:///fake-key", c.Model, c.APIKey}, "text", nil},
	} {
		if _, err := Extract(context.Background(), tc.config, tc.text, tc.image); err == nil || strings.Contains(err.Error(), "fake-key") {
			t.Fatal("invalid input accepted or key leaked")
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("invalid input triggered an HTTP request: calls=%d", calls.Load())
	}
}

func TestExtractProviderErrorsAreBoundedAndRedacted(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"unauthorized", `fake-key echoed raw response`, 401},
		{"rate limit", `fake-key echoed raw response`, 429},
		{"bad envelope", `fake-key echoed raw response`, 200},
		{"no choices", `{"choices":[]}`, 200},
		{"nonobject answer", `{"choices":[{"message":{"content":"null"}}]}`, 200},
		{"bad field type", `{"choices":[{"message":{"content":"{\"company\":[\"fake-key\"]}"}}]}`, 200},
		{"oversized response", strings.Repeat("x", maxResponseBytes+1), 200},
		{"redirect", `fake-key echoed raw response`, 307},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/redirect-target" {
					t.Error("credential-bearing request followed a redirect")
				}
				w.Header().Set("Location", "/redirect-target")
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.body)
			}))
			defer srv.Close()
			_, err := Extract(context.Background(), Config{srv.URL, "model", "fake-key"}, "job", nil)
			if err == nil || strings.Contains(err.Error(), "fake-key") || strings.Contains(err.Error(), "echoed raw response") {
				t.Fatal("provider failure accepted or response/key leaked")
			}
		})
	}
}

func TestExtractCancellationAndDeadline(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "deadline"}[deadline], func(t *testing.T) {
			started := make(chan struct{})
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.Copy(io.Discard, r.Body)
				close(started)
				select {
				case <-r.Context().Done():
				case <-time.After(2 * time.Second):
				}
			}))
			defer srv.Close()
			ctx, cancel := context.WithCancel(context.Background())
			if deadline {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
			}
			defer cancel()
			done := make(chan error, 1)
			go func() {
				_, err := Extract(ctx, Config{srv.URL, "model", "fake-key"}, "job", nil)
				done <- err
			}()
			select {
			case <-started:
			case <-time.After(3 * time.Second):
				t.Fatal("request did not start")
			}
			want := context.DeadlineExceeded
			if !deadline {
				want = context.Canceled
				cancel()
			}
			select {
			case err := <-done:
				if !errors.Is(err, want) || strings.Contains(err.Error(), "fake-key") {
					t.Fatalf("cancellation error=%v, want %v", err, want)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("Extract ignored context cancellation/deadline")
			}
		})
	}
}
