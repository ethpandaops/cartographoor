package inventory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoFetch_RejectsOversizedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		buf := make([]byte, 1024)
		for sent := 0; sent < maxDoraResponseBytes+1024; sent += len(buf) {
			if _, err := w.Write(buf); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	f := NewFetcher(logrus.NewEntry(logrus.New()))

	_, err := f.doFetch(context.Background(), srv.URL)

	require.Error(t, err, "expected an oversized response to be rejected instead of fully buffered")
}

func TestDoFetch_NormalResponseStillWorks(t *testing.T) {
	const body = `{"clients":[{"client_name":"lighthouse"}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	f := NewFetcher(logrus.NewEntry(logrus.New()))

	got, err := f.doFetch(context.Background(), srv.URL)

	require.NoError(t, err)
	assert.Equal(t, body, string(got))
}
