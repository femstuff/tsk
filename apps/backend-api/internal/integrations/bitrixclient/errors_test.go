package bitrixclient

import "testing"

func TestHasTaskScope(t *testing.T) {
	t.Parallel()

	cases := []struct {
		scope string
		want  bool
	}{
		{"task,user", true},
		{"crm,tasks", true},
		{"crm,tasks_extended,tasksmobile", true},
		{"app", true},
		{"", true},
		{"crm,user", false},
	}
	for _, tc := range cases {
		if got := HasTaskScope(tc.scope); got != tc.want {
			t.Fatalf("HasTaskScope(%q) = %v, want %v", tc.scope, got, tc.want)
		}
	}
}

func TestResolvePortalHost(t *testing.T) {
	t.Parallel()

	got := ResolvePortalHost(
		"b24-test.bitrix24.ru",
		"b24-fallback.bitrix24.ru",
		"https://b24-endpoint.bitrix24.ru/rest/",
		"oauth.bitrix.info",
	)
	if got != "b24-test.bitrix24.ru" {
		t.Fatalf("callback domain = %q", got)
	}

	got = ResolvePortalHost("", "b24-fallback.bitrix24.ru", "https://b24-endpoint.bitrix24.ru/rest/", "oauth.bitrix.info")
	if got != "b24-endpoint.bitrix24.ru" {
		t.Fatalf("client endpoint = %q", got)
	}
}
