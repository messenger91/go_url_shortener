package tests

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/save"
	"url-shortener/internal/lib/api"
	"url-shortener/internal/lib/random"
)

const (
	host = "localhost:8082"
)

func TestURLShortener_HappyPath(t *testing.T) {
	u := url.URL{
		Scheme: "http",
		Host:   host,
	}
	e := httpexpect.Default(t, u.String())

	e.POST("/url").
		WithJSON(save.Request{
			URL:   gofakeit.URL(),
			Alias: random.NewRandomString(10),
		}).
		WithBasicAuth("admin", "admin").
		Expect().
		Status(200).
		JSON().Object().
		ContainsKey("alias")
}

//nolint:funlen
func TestURLShortener_SaveRedirect(t *testing.T) {
	testCases := []struct {
		name   string
		url    string
		alias  string
		error  string
		status int
	}{
		{
			name:   "Valid URL",
			url:    gofakeit.URL(),
			alias:  gofakeit.Word() + gofakeit.Word(),
			status: http.StatusOK,
		},
		{
			name:   "Invalid URL",
			url:    "invalid_url",
			alias:  gofakeit.Word(),
			error:  "field URL is not a valid URL",
			status: http.StatusBadRequest,
		},
		{
			name:   "Empty Alias",
			url:    gofakeit.URL(),
			alias:  "",
			status: http.StatusOK,
		},
		// TODO: add more test cases
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			u := url.URL{
				Scheme: "http",
				Host:   host,
			}

			e := httpexpect.Default(t, u.String())

			// Save

			resp := e.POST("/url").
				WithJSON(save.Request{
					URL:   tc.url,
					Alias: tc.alias,
				}).
				WithBasicAuth("admin", "admin").
				Expect().Status(tc.status).
				JSON().Object()

			if tc.error != "" {
				resp.NotContainsKey("alias")

				resp.Value("error").String().IsEqual(tc.error)

				return
			}

			alias := tc.alias

			if tc.alias != "" {
				resp.Value("alias").String().IsEqual(tc.alias)
			} else {
				resp.Value("alias").String().NotEmpty()

				alias = resp.Value("alias").String().Raw()
			}

			// Redirect

			testRedirect(t, alias, tc.url)
		})
	}
}

func testRedirect(t *testing.T, alias string, urlToRedirect string) {
	u := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   alias,
	}

	redirectedToURL, err := api.GetRedirect(u.String())
	require.NoError(t, err)

	require.Equal(t, urlToRedirect, redirectedToURL)
}

func TestURLShortener_Stats(t *testing.T) {
	targetURL := gofakeit.URL()
	alias := random.NewRandomString(10)

	u := url.URL{
		Scheme: "http",
		Host:   host,
	}
	e := httpexpect.Default(t, u.String())

	e.POST("/url").
		WithJSON(save.Request{
			URL:   targetURL,
			Alias: alias,
		}).
		WithBasicAuth("admin", "admin").
		Expect().
		Status(http.StatusOK)

	// the redirect above must be counted as a visit
	testRedirect(t, alias, targetURL)

	stats := e.GET("/url/"+alias+"/stats").
		WithBasicAuth("admin", "admin").
		Expect().
		Status(http.StatusOK).
		JSON().Object()

	stats.Value("alias").String().IsEqual(alias)
	stats.Value("url").String().IsEqual(targetURL)
	stats.Value("visits").Number().IsEqual(1)

	e.GET("/url/"+random.NewRandomString(10)+"/stats").
		WithBasicAuth("admin", "admin").
		Expect().
		Status(http.StatusNotFound)
}

func TestURLShortener_Health(t *testing.T) {
	u := url.URL{
		Scheme: "http",
		Host:   host,
	}
	e := httpexpect.Default(t, u.String())

	e.GET("/health").
		Expect().
		Status(http.StatusOK).
		JSON().Object().
		Value("status").String().IsEqual("OK")
}

func TestURLShortener_Expired(t *testing.T) {
	alias := random.NewRandomString(10)

	u := url.URL{
		Scheme: "http",
		Host:   host,
	}
	e := httpexpect.Default(t, u.String())

	e.POST("/url").
		WithJSON(save.Request{
			URL:   gofakeit.URL(),
			Alias: alias,
			TTL:   1, // one second
		}).
		WithBasicAuth("admin", "admin").
		Expect().
		Status(http.StatusOK)

	// wait for the link to expire
	time.Sleep(2 * time.Second)

	e.GET("/" + alias).
		Expect().
		Status(http.StatusGone).
		JSON().Object().
		Value("error").String().IsEqual("url expired")

	// an expired link is still visible in the stats
	stats := e.GET("/url/"+alias+"/stats").
		WithBasicAuth("admin", "admin").
		Expect().
		Status(http.StatusOK).
		JSON().Object()

	stats.Value("visits").Number().IsEqual(0)
	stats.Value("expires_at").String().NotEmpty()
}
