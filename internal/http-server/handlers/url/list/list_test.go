package list_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/list"
	"url-shortener/internal/http-server/handlers/url/list/mocks"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/storage"
)

func TestListHandler(t *testing.T) {
	cases := []struct {
		name      string
		items     []storage.URLListItem
		status    int
		listError error
	}{
		{
			name: "Success",
			items: []storage.URLListItem{
				{URL: "https://example.com/2", Alias: "newest", Visits: 7},
				{
					URL:       "https://example.com/1",
					Alias:     "older",
					Visits:    3,
					ExpiresAt: time.Now().Add(time.Hour),
				},
			},
			status: http.StatusOK,
		},
		{
			name:      "List error",
			status:    http.StatusInternalServerError,
			listError: errors.New("unexpected error"),
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlListerMock := mocks.NewURLLister(t)

			urlListerMock.On("ListURLs").Return(tc.items, tc.listError).Once()

			handler := list.New(slogdiscard.NewDiscardLogger(), urlListerMock)

			req, err := http.NewRequest(http.MethodGet, "/url", nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, tc.status, rr.Code)

			var resp list.Response

			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

			require.Len(t, resp.Items, len(tc.items))
			for i, want := range tc.items {
				require.Equal(t, want.Alias, resp.Items[i].Alias)
				require.Equal(t, want.URL, resp.Items[i].URL)
				require.Equal(t, want.Visits, resp.Items[i].Visits)

				if want.ExpiresAt.IsZero() {
					require.Nil(t, resp.Items[i].ExpiresAt)
				} else {
					require.NotNil(t, resp.Items[i].ExpiresAt)
				}
			}
		})
	}
}
