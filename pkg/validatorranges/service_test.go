package validatorranges

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
	"github.com/ethpandaops/cartographoor/pkg/storage/s3"
	"github.com/sirupsen/logrus"
)

func TestStripRepoPrefix(t *testing.T) {
	tests := []struct {
		name        string
		networkName string
		repo        string
		want        string
	}{
		{
			name:        "prefix matches the repo name and is stripped",
			networkName: "glamsterdam-devnet-7",
			repo:        "ethpandaops/glamsterdam-devnets",
			want:        "devnet-7",
		},
		{
			name:        "another repo not in the old hardcoded list still strips correctly",
			networkName: "peerdas-devnet-7",
			repo:        "ethpandaops/peerdas-devnets",
			want:        "devnet-7",
		},
		{
			name:        "repo previously covered by the old hardcoded list still works",
			networkName: "fusaka-devnet-5",
			repo:        "ethpandaops/fusaka-devnets",
			want:        "devnet-5",
		},
		{
			name:        "network name has no matching repo prefix, left unchanged",
			networkName: "mainnet",
			repo:        "ethpandaops/ansible",
			want:        "mainnet",
		},
		{
			name:        "malformed repo path, left unchanged",
			networkName: "glamsterdam-devnet-7",
			repo:        "not-a-valid-repo-path",
			want:        "glamsterdam-devnet-7",
		},
		{
			name:        "empty repo, left unchanged",
			networkName: "glamsterdam-devnet-7",
			repo:        "",
			want:        "glamsterdam-devnet-7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripRepoPrefix(tt.networkName, tt.repo)
			if got != tt.want {
				t.Errorf("stripRepoPrefix(%q, %q) = %q, want %q", tt.networkName, tt.repo, got, tt.want)
			}
		})
	}
}

const testInventory = `[lighthouse_geth]
node1 ansible_host=1.2.3.4 validator_start=0 validator_end=8
`

// TestGenerateValidatorRanges_ReturnsErrorOnUploadFailure verifies that a
// network whose upload to S3 fails causes GenerateValidatorRanges to return
// a non-nil error, instead of the run being reported as fully successful.
//
// The fetcher has no injection seam, so processNetwork also attempts a real
// lookup against the ethpandaops fallback repository for this network. That
// lookup fails fast (the repository does not exist) and is non-fatal by
// design; the additional source below is what actually feeds this test.
func TestGenerateValidatorRanges_ReturnsErrorOnUploadFailure(t *testing.T) {
	inventorySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(testInventory))
	}))
	defer inventorySrv.Close()

	log := logrus.New()

	storage, err := s3.NewProvider(log, s3.Config{
		BucketName:     "test-bucket",
		Region:         "us-east-1",
		Endpoint:       "http://127.0.0.1:1", // nothing listens here, upload always fails
		ForcePathStyle: true,
		AccessKey:      "x",
		SecretKey:      "x",
	})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	cfg := &Config{
		AdditionalSources: map[string][]SourceConfig{
			"test-network": {
				{URL: inventorySrv.URL, Name: "test-source"},
			},
		},
	}

	svc := NewService(storage, cfg, log)

	networks := map[string]discovery.Network{
		"test-network": {Name: "test-network", Repository: ""},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err = svc.GenerateValidatorRanges(ctx, networks)

	if err == nil {
		t.Fatal("expected GenerateValidatorRanges to return an error when a network fails to upload, got nil")
	}
}

// TestGenerateValidatorRanges_NoErrorWhenNothingToUpload verifies that
// networks with no discoverable validator ranges (the normal, expected case
// for most networks) do not cause the run to fail.
func TestGenerateValidatorRanges_NoErrorWhenNothingToUpload(t *testing.T) {
	log := logrus.New()

	storage, err := s3.NewProvider(log, s3.Config{
		BucketName: "test-bucket",
		Region:     "us-east-1",
	})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	svc := NewService(storage, &Config{}, log)

	networks := map[string]discovery.Network{
		"mainnet": {Name: "mainnet", Repository: ""},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := svc.GenerateValidatorRanges(ctx, networks); err != nil {
		t.Fatalf("expected no error when a network simply has no ranges to publish, got %v", err)
	}
}
