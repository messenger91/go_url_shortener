package sqlite_test

import (
	"path/filepath"
	"testing"

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

	id, err := st.SaveURL(testURL, testAlias)
	require.NoError(t, err)
	require.NotZero(t, id)

	gotURL, err := st.GetURL(testAlias)
	require.NoError(t, err)
	require.Equal(t, testURL, gotURL)

	_, err = st.SaveURL("https://example.com/other", testAlias)
	require.ErrorIs(t, err, storage.ErrURLExists)

	// visits are recorded per redirect; unknown aliases are allowed and
	// become orphans when the URL is deleted
	require.NoError(t, st.RecordVisit(testAlias))
	require.NoError(t, st.RecordVisit(testAlias))
	require.NoError(t, st.RecordVisit("unknown_alias"))

	count, err := st.GetVisitsCount(testAlias)
	require.NoError(t, err)
	require.Equal(t, int64(2), count)

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
