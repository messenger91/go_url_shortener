package sqlite_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"url-shortener/internal/storage"
	"url-shortener/internal/storage/sqlite"
)

func TestStorage(t *testing.T) {
	st, err := sqlite.Connect(filepath.Join(t.TempDir(), "storage.db"))
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

	require.NoError(t, st.DeleteURL(testAlias))

	_, err = st.GetURL(testAlias)
	require.ErrorIs(t, err, storage.ErrURLNotFound)

	err = st.DeleteURL(testAlias)
	require.ErrorIs(t, err, storage.ErrURLNotFound)
}
