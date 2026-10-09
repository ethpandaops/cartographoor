package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gh "github.com/google/go-github/v53/github"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetermineNetworkStatus covers how a network's status is derived from the
// kubernetes and kubernetes-archive directory lookups. A network must only be
// marked active or inactive on a positive 200 response. Any other response,
// including a transient 403 rate limit or a 5xx outage, must leave the status
// unknown rather than laundering the error into a false active/inactive.
func TestDetermineNetworkStatus(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)

	const okBody = `[{"name":"file.yaml","type":"file"}]`

	cases := []struct {
		name          string
		kubeStatus    int
		archiveStatus int
		want          string
	}{
		{"present in kubernetes is active", http.StatusOK, http.StatusNotFound, "active"},
		{"present in archive is inactive", http.StatusNotFound, http.StatusOK, "inactive"},
		{"absent in both is unknown", http.StatusNotFound, http.StatusNotFound, "unknown"},
		{"rate limit does not imply active", http.StatusForbidden, http.StatusForbidden, "unknown"},
		{"server error does not imply active", http.StatusInternalServerError, http.StatusInternalServerError, "unknown"},
		{"unauthorized does not imply active", http.StatusUnauthorized, http.StatusUnauthorized, "unknown"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var code int

				switch {
				case strings.Contains(r.URL.Path, kubernetesArchiveDir):
					code = tc.archiveStatus
				case strings.Contains(r.URL.Path, kubernetesDir):
					code = tc.kubeStatus
				default:
					// Any secondary lookups (for example config values on the
					// active path) are not under test.
					code = http.StatusNotFound
				}

				w.WriteHeader(code)

				if code == http.StatusOK {
					_, _ = w.Write([]byte(okBody))
				}
			}))
			defer ts.Close()

			provider, err := NewProvider(log, nil)
			require.NoError(t, err)

			client := gh.NewClient(&http.Client{Transport: &mockTransport{URL: ts.URL}})

			status, _, _ := provider.determineNetworkStatus(
				context.Background(), client, "ethpandaops", "devnets", "devnet-x")

			assert.Equal(t, tc.want, status)
		})
	}
}
