package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// type ProgressReporter func(done chan int64, path string, total int64)

func DownloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// write to a tmp file first to avoid leaving a corrupt file on failure
	tmpPath := filepath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		// clean up the partial tmp file
		os.Remove(tmpPath)
		return err
	}

	// close before rename so the file is fully flushed
	out.Close()

	// atomically move the tmp file to the final destination
	return os.Rename(tmpPath, filepath)
}

func PrintDownloadPercent(done chan int64, path string, total int64) {
	var stop bool = false

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			stop = true
		case <-ticker.C:
			file, err := os.Open(path)
			if err != nil {
				log.Error().Err(err).Msg("failed to open the file")
			}

			fi, err := file.Stat()
			if err != nil {
				log.Error().Err(err).Msg("failed to get file stats")
			}

			size := fi.Size()
			if size == 0 {
				size = 1
			}

			var percent = fmt.Sprintf("%.2f", float64(size)/float64(total)*100)

			log.Info().Str("percent", percent).Msg("downloading")
			fmt.Print("\033[A")
		}

		if stop {
			break
		}
	}
}

func DownloadFileWithProgress(url string, dest string) error {
	headResp, err := http.Head(url)
	if err != nil {
		log.Error().Err(err).Msg("failed to get head")
		return err
	}
	defer headResp.Body.Close()

	_, err = strconv.Atoi(headResp.Header.Get("Content-Length"))
	if err != nil {
		log.Error().Err(err).Msg("failed to get content length")
		return err
	}

	eTagHeader := headResp.Header.Get("ETag")
	eTag := strings.Trim(eTagHeader, "\"")
	eTagFilename := ChangeExt(dest, "etag")
	if FileExists(eTagFilename) {
		readEtag, err := os.ReadFile(eTagFilename)
		if err != nil {
			log.Error().Err(err).Msg("failed to read etag file. will try to create a new one")
		} else {
			if string(readEtag) == eTag {
				// same file. exit
				log.Info().Str("filename", dest).Msg("no file changed")
				return nil
			}
		}
	}

	// write to a tmp file first to avoid leaving a corrupt file on failure
	tmpPath := dest + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		log.Error().Err(err).Msg("failed to create tmp file")
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		// clean up the partial tmp file
		os.Remove(tmpPath)
		log.Error().Err(err).Msg("failed to get")
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		// clean up the partial tmp file
		os.Remove(tmpPath)
		log.Error().Err(err).Msg("failed to copy")
		return err
	}

	// close before rename so the file is fully flushed
	out.Close()

	// atomically move the tmp file to the final destination
	err = os.Rename(tmpPath, dest)
	if err != nil {
		os.Remove(tmpPath)
		log.Error().Err(err).Msg("failed to rename tmp file to destination")
		return err
	}

	// write the etag file only after the db file is successfully in place
	eTagFile, err := os.Create(eTagFilename)
	if err != nil {
		log.Error().Err(err).Msg("failed to create etag file")
		return err
	}
	defer eTagFile.Close()

	_, err = eTagFile.WriteString(eTag)
	if err != nil {
		log.Error().Err(err).Msg("failed to write etag file")
		return err
	}

	return nil
}

// DownloadMaxMindDb downloads a database directly from MaxMind's API using
// HTTP Basic Auth. The response is a tar.gz archive containing the .mmdb file.
// It uses ETag-based caching to skip re-downloads when the database hasn't changed.
func DownloadMaxMindDb(accountID, licenseKey, editionID, dest string) error {
	if licenseKey == "" {
		return fmt.Errorf("MaxMind license_key is required for direct download")
	}
	if accountID == "" {
		return fmt.Errorf("MaxMind account_id is required when license_key is set")
	}
	if editionID == "" {
		return fmt.Errorf("MaxMind edition ID is required for direct download")
	}

	baseURL := fmt.Sprintf("https://download.maxmind.com/geoip/databases/%s/download?suffix=tar.gz", editionID)
	return downloadMaxMindDbFromURL(baseURL, accountID, licenseKey, editionID, dest)
}

// downloadMaxMindDbFromURL performs the actual download from a given URL.
// Separated from DownloadMaxMindDb to allow testing with httptest servers.
func downloadMaxMindDbFromURL(url, accountID, licenseKey, editionID, dest string) error {
	// Use a non-redirect client for HEAD so we get the ETag directly from
	// MaxMind without following the redirect to R2 (which strips auth headers).
	noRedirectClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	headReq, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create HEAD request: %w", err)
	}
	headReq.SetBasicAuth(accountID, licenseKey)

	headResp, err := noRedirectClient.Do(headReq)
	if err != nil {
		return fmt.Errorf("failed to HEAD MaxMind API: %w", err)
	}
	defer headResp.Body.Close()

	// Accept both 200 and 3xx as valid responses
	if headResp.StatusCode != http.StatusOK && (headResp.StatusCode < 300 || headResp.StatusCode >= 400) {
		return fmt.Errorf("MaxMind HEAD request failed with status %d", headResp.StatusCode)
	}

	eTagHeader := headResp.Header.Get("ETag")
	eTag := strings.Trim(eTagHeader, "\"")
	eTagFilename := ChangeExt(dest, "etag")

	if eTag != "" && FileExists(eTagFilename) {
		readEtag, err := os.ReadFile(eTagFilename)
		if err != nil {
			log.Error().Err(err).Msg("failed to read etag file, will re-download")
		} else if string(readEtag) == eTag {
			log.Info().Str("edition", editionID).Msg("no database change detected")
			return nil
		}
	}

	// GET request to download. MaxMind redirects to a Cloudflare R2 presigned
	// URL, so we need to follow redirects. The presigned URL contains auth in
	// query params, so no need to forward Basic Auth to the redirect target.
	getReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create GET request: %w", err)
	}
	getReq.SetBasicAuth(accountID, licenseKey)

	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		return fmt.Errorf("failed to GET MaxMind database: %w", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		return fmt.Errorf("MaxMind GET request failed with status %d", getResp.StatusCode)
	}

	// Write tar.gz to temp file
	tarGzTmp := dest + ".tar.gz.tmp"
	tarGzFile, err := os.Create(tarGzTmp)
	if err != nil {
		return fmt.Errorf("failed to create temp tar.gz file: %w", err)
	}

	_, err = io.Copy(tarGzFile, getResp.Body)
	tarGzFile.Close()
	if err != nil {
		os.Remove(tarGzTmp)
		return fmt.Errorf("failed to download tar.gz: %w", err)
	}

	// Extract .mmdb from tar.gz
	mmdbTmp := dest + ".tmp"
	err = extractMmdbFromTarGz(tarGzTmp, mmdbTmp)
	os.Remove(tarGzTmp)
	if err != nil {
		os.Remove(mmdbTmp)
		return fmt.Errorf("failed to extract mmdb from tar.gz: %w", err)
	}

	// Atomic rename
	err = os.Rename(mmdbTmp, dest)
	if err != nil {
		os.Remove(mmdbTmp)
		return fmt.Errorf("failed to rename mmdb to destination: %w", err)
	}

	// Write ETag
	err = os.WriteFile(eTagFilename, []byte(eTag), 0644)
	if err != nil {
		log.Error().Err(err).Msg("failed to write etag file")
	}

	log.Info().Str("edition", editionID).Str("dest", dest).Msg("MaxMind database downloaded successfully")
	return nil
}

// extractMmdbFromTarGz opens a tar.gz archive and extracts the first .mmdb file to destPath.
func extractMmdbFromTarGz(tarGzPath, destPath string) error {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		if filepath.Ext(header.Name) != ".mmdb" {
			continue
		}

		out, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}

		_, err = io.Copy(out, tr)
		out.Close()
		if err != nil {
			return fmt.Errorf("failed to extract mmdb: %w", err)
		}

		return nil
	}

	return fmt.Errorf("no .mmdb file found in archive")
}
