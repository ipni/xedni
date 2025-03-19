package xedni_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ipni/xedni"
	"github.com/stretchr/testify/require"
)

func TestSampleHandler(t *testing.T) {
	for _, test := range []struct {
		name           string
		url            string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Invalid Provider ID",
			url:            "/ipni/v0/sample/invalidProviderID/validContextID",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"invalid provider ID"}`,
		},
		{
			name:           "Invalid Context ID",
			url:            "/ipni/v0/sample/12D3KooWKTMKoNRJUwdGjuoY3FdtXzARas9UczGsPLw2MgPaLCnh/invalidContextID",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"invalid Context ID"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.url, nil)
			rec := httptest.NewRecorder()

			subject, err := xedni.New(
				xedni.WithStorePath(t.TempDir()),
				xedni.WithDelegateIndexer(noopStore{}))
			require.NoError(t, err)
			mux := subject.ServeMux()
			mux.ServeHTTP(rec, req)

			res := rec.Result()
			defer res.Body.Close()

			if res.StatusCode != test.expectedStatus {
				t.Errorf("expected status %d, got %d", test.expectedStatus, res.StatusCode)
			}

			body := rec.Body.String()
			if !strings.Contains(body, test.expectedBody) {
				t.Errorf("expected body to contain %q, got %q", test.expectedBody, body)
			}
		})
	}
}
