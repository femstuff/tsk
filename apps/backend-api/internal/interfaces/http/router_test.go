package httpapi

import (
	"net/http"
	"testing"
)

func TestIsBusinessRequest(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		method         string
		path           string
		requestSource  string
		wantBusiness   bool
	}{
		{
			name:         "counts product list endpoint",
			method:       http.MethodGet,
			path:         "/api/v1/document-jobs",
			wantBusiness: true,
		},
		{
			name:         "excludes metrics scrape",
			method:       http.MethodGet,
			path:         "/metrics",
			wantBusiness: false,
		},
		{
			name:         "excludes health checks",
			method:       http.MethodGet,
			path:         "/api/v1/health",
			wantBusiness: false,
		},
		{
			name:         "excludes admin polling",
			method:       http.MethodGet,
			path:         "/api/v1/document-jobs",
			requestSource: requestSourceAdminPoll,
			wantBusiness: false,
		},
		{
			name:         "excludes cors preflight",
			method:       http.MethodOptions,
			path:         "/api/v1/document-jobs",
			wantBusiness: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptestRequest(t, tc.method, tc.path)
			if tc.requestSource != "" {
				req.Header.Set(requestSourceHeader, tc.requestSource)
			}

			got := isBusinessRequest(req, normalizePath(tc.path))
			if got != tc.wantBusiness {
				t.Fatalf("isBusinessRequest(%s %s) = %v, want %v", tc.method, tc.path, got, tc.wantBusiness)
			}
		})
	}
}

func httptestRequest(t *testing.T, method string, path string) *http.Request {
	t.Helper()

	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	return req
}
