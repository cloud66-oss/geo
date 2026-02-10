package utils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// createTestTarGz creates a tar.gz archive containing a single .mmdb file with the given content.
func createTestTarGz(t *testing.T, mmdbContent []byte, innerPath string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	err := tw.WriteHeader(&tar.Header{
		Name: innerPath,
		Mode: 0644,
		Size: int64(len(mmdbContent)),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = tw.Write(mmdbContent)
	if err != nil {
		t.Fatal(err)
	}

	tw.Close()
	gw.Close()
	return buf.Bytes()
}

func TestExtractMmdbFromTarGz(t *testing.T) {
	mmdbContent := []byte("fake-mmdb-data-for-testing")
	tarGzData := createTestTarGz(t, mmdbContent, "GeoLite2-City_20240101/GeoLite2-City.mmdb")

	tmpDir := t.TempDir()
	tarGzPath := filepath.Join(tmpDir, "test.tar.gz")
	destPath := filepath.Join(tmpDir, "output.mmdb")

	err := os.WriteFile(tarGzPath, tarGzData, 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = extractMmdbFromTarGz(tarGzPath, destPath)
	if err != nil {
		t.Fatalf("extractMmdbFromTarGz failed: %v", err)
	}

	extracted, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(extracted, mmdbContent) {
		t.Errorf("extracted content mismatch: got %q, want %q", extracted, mmdbContent)
	}
}

func TestExtractMmdbFromTarGz_NoMmdb(t *testing.T) {
	// Create a tar.gz with a non-mmdb file
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	content := []byte("not a database")
	tw.WriteHeader(&tar.Header{
		Name: "README.txt",
		Mode: 0644,
		Size: int64(len(content)),
	})
	tw.Write(content)
	tw.Close()
	gw.Close()

	tmpDir := t.TempDir()
	tarGzPath := filepath.Join(tmpDir, "test.tar.gz")
	destPath := filepath.Join(tmpDir, "output.mmdb")

	os.WriteFile(tarGzPath, buf.Bytes(), 0644)

	err := extractMmdbFromTarGz(tarGzPath, destPath)
	if err == nil {
		t.Fatal("expected error when no .mmdb file in archive")
	}
}

func TestDownloadMaxMindDb(t *testing.T) {
	mmdbContent := []byte("test-mmdb-database-content")
	tarGzData := createTestTarGz(t, mmdbContent, "GeoLite2-City_20240101/GeoLite2-City.mmdb")

	expectedAccountID := "123456"
	expectedLicenseKey := "test-license-key"
	testETag := "abc123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate Basic Auth
		user, pass, ok := r.BasicAuth()
		if !ok || user != expectedAccountID || pass != expectedLicenseKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("ETag", `"`+testETag+`"`)

		if r.Method == "HEAD" {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.Header().Set("Content-Type", "application/gzip")
		w.Write(tarGzData)
	}))
	defer server.Close()

	// Temporarily override the download URL by using a custom function
	// Since DownloadMaxMindDb hardcodes the URL, we test extractMmdbFromTarGz + the server separately
	// Instead, test the full flow using a helper that accepts a base URL
	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "GeoLite2-City.mmdb")

	// Test with the mock server - we need to call the internal logic
	// Since we can't override the URL in DownloadMaxMindDb, test the pieces
	t.Run("full flow with mock server", func(t *testing.T) {
		err := downloadMaxMindDbFromURL(server.URL, expectedAccountID, expectedLicenseKey, "GeoLite2-City", dest)
		if err != nil {
			t.Fatalf("download failed: %v", err)
		}

		extracted, err := os.ReadFile(dest)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(extracted, mmdbContent) {
			t.Errorf("content mismatch: got %q, want %q", extracted, mmdbContent)
		}

		// Verify ETag file was written
		etagFile := ChangeExt(dest, "etag")
		etagContent, err := os.ReadFile(etagFile)
		if err != nil {
			t.Fatal(err)
		}
		if string(etagContent) != testETag {
			t.Errorf("etag mismatch: got %q, want %q", etagContent, testETag)
		}
	})

	t.Run("skip download on matching etag", func(t *testing.T) {
		// The etag file already exists from the previous test run
		// Running again should skip the download
		err := downloadMaxMindDbFromURL(server.URL, expectedAccountID, expectedLicenseKey, "GeoLite2-City", dest)
		if err != nil {
			t.Fatalf("second download failed: %v", err)
		}
	})

	t.Run("bad credentials", func(t *testing.T) {
		badDest := filepath.Join(tmpDir, "bad.mmdb")
		err := downloadMaxMindDbFromURL(server.URL, "wrong", "wrong", "GeoLite2-City", badDest)
		if err == nil {
			t.Fatal("expected error with bad credentials")
		}
	})
}

func TestDownloadMaxMindDb_ValidationErrors(t *testing.T) {
	tests := []struct {
		name       string
		accountID  string
		licenseKey string
		editionID  string
	}{
		{"empty license key", "123", "", "GeoLite2-City"},
		{"empty account ID", "", "key", "GeoLite2-City"},
		{"empty edition ID", "123", "key", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DownloadMaxMindDb(tt.accountID, tt.licenseKey, tt.editionID, "/tmp/test.mmdb")
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
