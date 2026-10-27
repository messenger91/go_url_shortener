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

	// the default breakdown covers 30 days, the redirect landed today
	byDay := stats.Value("visits_by_day").Array()
	byDay.Length().IsEqual(30)
	byDay.Value(0).Object().Value("count").Number().IsEqual(0)
	byDay.Value(29).Object().Value("count").Number().IsEqual(1)

	stats7 := e.GET("/url/"+alias+"/stats").
		WithQuery("days", 7).
		WithBasicAuth("admin", "admin").
		Expect().
		Status(http.StatusOK).
		JSON().Object()

	stats7.Value("visits_by_day").Array().Length().IsEqual(7)

	e.GET("/url/"+alias+"/stats").
		WithQuery("days", "abc").
		WithBasicAuth("admin", "admin").
		Expect().
		Status(http.StatusBadRequest)

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

func TestURLShortener_UI(t *testing.T) {
	u := url.URL{
		Scheme: "http",
		Host:   host,
	}
	e := httpexpect.Default(t, u.String())

	index := e.GET("/").Expect().Status(http.StatusOK)
	index.Header("Content-Type").Contains("text/html")
	index.Body().Contains("URL Shortener")

	e.GET("/static/style.css").Expect().Status(http.StatusOK)
	e.GET("/static/app.js").Expect().Status(http.StatusOK)
}

//nolint:funlen
func TestURLShortener_LoginAndCRUD(t *testing.T) {
	u := url.URL{
		Scheme: "http",
		Host:   host,
	}
	e := httpexpect.Default(t, u.String())

	// anonymous requests get a JSON 401 without WWW-Authenticate
	e.GET("/url").
		Expect().
		Status(http.StatusUnauthorized).
		JSON().Object().
		Value("error").String().IsEqual("unauthorized")

	// wrong credentials are rejected
	e.POST("/login").
		WithJSON(map[string]string{"user": "admin", "password": "wrong"}).
		Expect().
		Status(http.StatusUnauthorized)

	// login with the default admin:admin issues a session cookie
	session := e.POST("/login").
		WithJSON(map[string]string{"user": "admin", "password": "admin"}).
		Expect().
		Status(http.StatusOK).
		Cookie("session").Value().Raw()

	targetURL := gofakeit.URL()
	alias := random.NewRandomString(10)

	// create via session cookie
	e.POST("/url").
		WithCookie("session", session).
		WithJSON(save.Request{
			URL:   targetURL,
			Alias: alias,
		}).
		Expect().
		Status(http.StatusOK)

	// the list contains the new link with zero visits
	items := e.GET("/url").
		WithCookie("session", session).
		Expect().
		Status(http.StatusOK).
		JSON().Object().Value("items").Array()

	found := false
	for i := 0; i < int(items.Length().Raw()); i++ {
		it := items.Value(i).Object()
		if it.Value("alias").String().Raw() == alias {
			found = true
			it.Value("url").String().IsEqual(targetURL)
			it.Value("visits").Number().IsEqual(0)
		}
	}
	require.True(t, found)

	// update retargets the link and gives it a TTL
	newTarget := gofakeit.URL()
	e.PUT("/url/"+alias).
		WithCookie("session", session).
		WithJSON(map[string]any{"url": newTarget, "ttl_seconds": 3600}).
		Expect().
		Status(http.StatusOK)

	testRedirect(t, alias, newTarget)

	e.GET("/url/"+alias+"/stats").
		WithCookie("session", session).
		Expect().
		Status(http.StatusOK).
		JSON().Object().
		Value("expires_at").String().NotEmpty()

	// delete
	e.DELETE("/url/"+alias).
		WithCookie("session", session).
		Expect().
		Status(http.StatusOK)

	e.GET("/" + alias).
		Expect().
		Status(http.StatusNotFound)

	// logout clears the session
	e.POST("/logout").
		Expect().
		Status(http.StatusOK)
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
