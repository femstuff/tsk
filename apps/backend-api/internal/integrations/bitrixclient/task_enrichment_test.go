package bitrixclient

import (
	"encoding/json"
	"testing"
)

func TestParseTaskCommentsResult(t *testing.T) {
	t.Parallel()

	raw := `[
		{
			"ID": "10",
			"AUTHOR_ID": "503",
			"AUTHOR_NAME": "Maria Johnson",
			"POST_DATE": "2025-10-06T17:02:43+02:00",
			"POST_MESSAGE": "Проверьте вложение",
			"ATTACHED_OBJECTS": {
				"55": {
					"ATTACHMENT_ID": "55",
					"NAME": "scan.png",
					"SIZE": "1024",
					"DOWNLOAD_URL": "/bitrix/tools/disk/uf.php?attachedId=55"
				}
			}
		}
	]`

	comments, err := parseTaskCommentsResult(json.RawMessage(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 1 {
		t.Fatalf("comments = %+v", comments)
	}
	if comments[0].AuthorName != "Maria Johnson" {
		t.Fatalf("author = %q", comments[0].AuthorName)
	}
	if len(comments[0].Files) != 1 || comments[0].Files[0].Name != "scan.png" {
		t.Fatalf("files = %+v", comments[0].Files)
	}
}

func TestParseBitrixTaskIDInt(t *testing.T) {
	t.Parallel()

	cases := map[string]int{
		"8017":    8017,
		" 123 ":   123,
		"8017.0":  8017,
	}
	for raw, want := range cases {
		got, err := parseBitrixTaskIDInt(raw)
		if err != nil {
			t.Fatalf("parseBitrixTaskIDInt(%q): %v", raw, err)
		}
		if got != want {
			t.Fatalf("parseBitrixTaskIDInt(%q) = %d, want %d", raw, got, want)
		}
	}
}

func TestAbsolutizeBitrixURL(t *testing.T) {
	t.Parallel()

	got := absolutizeBitrixURL("/bitrix/tools/disk/uf.php", "demo.bitrix24.ru")
	if got != "https://demo.bitrix24.ru/bitrix/tools/disk/uf.php" {
		t.Fatalf("got %q", got)
	}
}
