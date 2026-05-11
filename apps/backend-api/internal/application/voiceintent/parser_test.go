package voiceintent

import "testing"

func TestParse_DealTitleOnNextStage(t *testing.T) {
	cases := []struct {
		text       string
		wantAction string
		wantTitle  string
	}{
		{
			text:       "Перемести сделку тест на следующий этап",
			wantAction: ActionMoveNext,
			wantTitle:  "тест",
		},
		{
			text:       "переведи сделку моя сделка на следующий этап",
			wantAction: ActionMoveNext,
			wantTitle:  "моя сделка",
		},
		{
			text:       "сделку тест следующий этап",
			wantAction: ActionMoveNext,
			wantTitle:  "тест",
		},
	}
	for _, tc := range cases {
		got := Parse(tc.text, 0)
		if got.Action != tc.wantAction {
			t.Fatalf("%q: action=%q want %q", tc.text, got.Action, tc.wantAction)
		}
		if got.DealTitle != tc.wantTitle {
			t.Fatalf("%q: DealTitle=%q want %q", tc.text, got.DealTitle, tc.wantTitle)
		}
	}
}
