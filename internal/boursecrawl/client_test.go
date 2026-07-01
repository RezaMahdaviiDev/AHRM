package boursecrawl_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ahrm/internal/boursecrawl"
)

func TestFetchLatestNotice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><div>پیام ناظر: نماد به علت افشای اطلاعات با اهمیت متوقف شد. تاریخ 1405/04/12 09:15</div></body></html>`))
	}))
	defer srv.Close()

	client := boursecrawl.NewClient(srv.URL+"?symbol={symbol}", "test-agent")
	notice, err := client.FetchLatestNotice(context.Background(), "خساپا")
	if err != nil {
		t.Fatal(err)
	}
	if notice.Reason == "" {
		t.Fatal("reason must be extracted")
	}
	if notice.PublishedAt != "1405/04/12 09:15" {
		t.Fatalf("publishedAt=%q", notice.PublishedAt)
	}
}

func TestFetchLatestNoticeNoReason(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><div>بدون پیام توقف</div></body></html>`))
	}))
	defer srv.Close()

	client := boursecrawl.NewClient(srv.URL, "")
	_, err := client.FetchLatestNotice(context.Background(), "خساپا")
	if err == nil {
		t.Fatal("expected error when no halt message exists")
	}
}
