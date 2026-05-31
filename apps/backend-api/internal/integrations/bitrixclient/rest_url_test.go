package bitrixclient

import (
	"net/url"
	"strings"
	"testing"
)

func TestRestBaseURL(t *testing.T) {
	t.Parallel()

	got := RestBaseURL("https://b24-test.bitrix24.ru/", "https://b24-test.bitrix24.ru/rest/")
	want := "https://b24-test.bitrix24.ru/rest/"
	if got != want {
		t.Fatalf("RestBaseURL(client_endpoint) = %q, want %q", got, want)
	}

	got = RestBaseURL("b24-test.bitrix24.ru", "")
	want = "https://b24-test.bitrix24.ru/rest/"
	if got != want {
		t.Fatalf("RestBaseURL(portal) = %q, want %q", got, want)
	}
}

func TestRestAPIV3Base(t *testing.T) {
	t.Parallel()

	got := RestAPIV3Base("https://b24-test.bitrix24.ru/rest/1/abc/")
	want := "https://b24-test.bitrix24.ru/rest/api/1/abc"
	if got != want {
		t.Fatalf("RestAPIV3Base(webhook) = %q, want %q", got, want)
	}

	got = RestAPIV3Base("https://b24-test.bitrix24.ru/rest/")
	want = "https://b24-test.bitrix24.ru/rest/api"
	if got != want {
		t.Fatalf("RestAPIV3Base(oauth base) = %q, want %q", got, want)
	}
}

func TestTokenRESTMethodURL(t *testing.T) {
	t.Parallel()

	client := NewTokenREST("b24-test.bitrix24.ru", "", "token-abc", nil)
	raw := client.methodURL("tasks.task.list")
	if strings.Contains(raw, ".json") {
		t.Fatalf("oauth method URL must not contain .json: %s", raw)
	}
	if !strings.HasPrefix(raw, "https://b24-test.bitrix24.ru/rest/tasks.task.list?auth=") {
		t.Fatalf("unexpected oauth method URL: %s", raw)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("auth") != "token-abc" {
		t.Fatalf("auth query = %q", parsed.Query().Get("auth"))
	}
}
