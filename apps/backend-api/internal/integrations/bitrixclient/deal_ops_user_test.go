package bitrixclient

import "testing"

func TestUserDisplayNameFromRow(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		row  map[string]any
		want string
	}{
		{
			name: "first and last",
			row:  map[string]any{"NAME": "Иван", "LAST_NAME": "Петров"},
			want: "Петров Иван",
		},
		{
			name: "full fio",
			row:  map[string]any{"NAME": "Иван", "LAST_NAME": "Петров", "SECOND_NAME": "Сергеевич"},
			want: "Петров Иван Сергеевич",
		},
		{
			name: "formatted",
			row:  map[string]any{"FORMATTED_NAME": "Петров Иван", "NAME": "Иван"},
			want: "Петров Иван",
		},
		{
			name: "last only",
			row:  map[string]any{"LAST_NAME": "Сидоров"},
			want: "Сидоров",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := userDisplayNameFromRow(tc.row); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
