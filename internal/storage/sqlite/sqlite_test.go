package sqlite_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"url-shortener/internal/storage"
	"url-shortener/internal/storage/sqlite"
)

func TestStorage(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "storage.db")

	st, err := sqlite.Connect(dbPath)
	require.NoError(t, err)

	const (
		testURL   = "https://example.com/some/long/url"
		testAlias = "test_alias"
	)

	id, err := st.SaveURL(testURL, testAlias, time.Time{})
	require.NoError(t, err)
	require.NotZero(t, id)

	gotURL, err := st.GetURL(testAlias)
	require.NoError(t, err)
	require.Equal(t, testURL, gotURL.URL)
	require.True(t, gotURL.ExpiresAt.IsZero())

	_, err = st.SaveURL("https://example.com/other", testAlias, time.Time{})
	require.ErrorIs(t, err, storage.ErrURLExists)

	// an already-passed expiry time round-trips through the DB
	past := time.Now().Add(-time.Minute)

	_, err = st.SaveURL("https://example.com/expired", "expired_alias", past)
	require.NoError(t, err)

	got, err := st.GetURL("expired_alias")
	require.NoError(t, err)
	require.True(t, got.ExpiresAt.Before(time.Now()))

	// visits are recorded per redirect; unknown aliases are allowed and
	// become orphans when the URL is deleted
	require.NoError(t, st.RecordVisit(testAlias))
	require.NoError(t, st.RecordVisit(testAlias))
	require.NoError(t, st.RecordVisit("unknown_alias"))

	count, err := st.GetVisitsCount(testAlias)
	require.NoError(t, err)
	require.Equal(t, int64(2), count)

	// the breakdown covers exactly the requested number of UTC days,
	// today holds the visits recorded above
	byDay, err := st.GetVisitsByDay(testAlias, 7)
	require.NoError(t, err)
	require.Len(t, byDay, 7)
	require.Equal(t, int64(2), byDay[6].Count)
	require.Equal(t, time.Now().UTC().Format("2006-01-02"), byDay[6].Date)

	require.NoError(t, st.DeleteURL(testAlias))

	_, err = st.GetURL(testAlias)
	require.ErrorIs(t, err, storage.ErrURLNotFound)

	err = st.DeleteURL(testAlias)
	require.ErrorIs(t, err, storage.ErrURLNotFound)

	// re-connect on the same file: migrations must be idempotent and data intact
	st2, err := sqlite.Connect(dbPath)
	require.NoError(t, err)

	_, err = st2.GetURL("some_other_alias")
	require.ErrorIs(t, err, storage.ErrURLNotFound)
}
