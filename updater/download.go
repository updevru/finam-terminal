package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// downloadTimeout bounds a whole self-update download. Five minutes is
// generous for a ~15 MB binary even on a slow link, while still guaranteeing
// the operation cannot hang forever.
const downloadTimeout = 5 * time.Minute

// maxChecksumsBytes caps the checksums.txt download; the real file is a few
// hundred bytes.
const maxChecksumsBytes = 1 << 20 // 1 MiB

// downloadAsset streams a release asset to destPath and verifies its
// integrity before returning.
//
// Verification prefers the SHA256 published in the release's checksums.txt. If
// that asset is absent (releases predating it) or carries no line for this
// binary, the downloaded size is compared against the size GitHub reports for
// the asset instead. Either way a mismatch is an error.
//
// progress, when not nil, is called with the number of bytes written so far
// and the expected total; it is invoked from the calling goroutine.
//
// On any error the partially written file is removed, so the caller can be
// sure destPath either holds a fully verified binary or does not exist.
func downloadAsset(ctx context.Context, rel *Release, asset *Asset, destPath string, progress func(done, total int64)) error {
	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()

	// The expected checksum is fetched first: failing before a multi-megabyte
	// download is cheaper than failing after it.
	expectedSum := lookupChecksum(ctx, rel, asset.Name)

	sum, written, err := streamToFile(ctx, asset, destPath, progress)
	if err != nil {
		_ = os.Remove(destPath)
		return err
	}

	if err := verifyDownload(asset, expectedSum, sum, written); err != nil {
		_ = os.Remove(destPath)
		return err
	}
	return nil
}

// streamToFile downloads the asset into destPath, hashing the bytes as they
// are written, and returns the hex SHA256 and the number of bytes written.
func streamToFile(ctx context.Context, asset *Asset, destPath string, progress func(done, total int64)) (string, int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.DownloadURL, nil)
	if err != nil {
		return "", 0, fmt.Errorf("build download request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("download %s: %w", asset.Name, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("download %s: server returned %s", asset.Name, resp.Status)
	}

	total := asset.Size
	if total <= 0 {
		total = resp.ContentLength
	}

	file, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return "", 0, fmt.Errorf("create %s: %w", destPath, err)
	}

	digest := sha256.New()
	written, copyErr := copyWithProgress(io.MultiWriter(file, digest), resp.Body, total, progress)

	if closeErr := file.Close(); closeErr != nil && copyErr == nil {
		copyErr = fmt.Errorf("close %s: %w", destPath, closeErr)
	}
	if copyErr != nil {
		return "", written, copyErr
	}
	return hex.EncodeToString(digest.Sum(nil)), written, nil
}

// copyWithProgress copies src into dst in chunks, reporting the running byte
// count after each one. dst is the io.MultiWriter fanning the stream out to
// both the destination file and the SHA256 digest.
func copyWithProgress(dst io.Writer, src io.Reader, total int64, progress func(done, total int64)) (int64, error) {
	const chunkSize = 64 << 10 // 64 KiB
	buf := make([]byte, chunkSize)

	var written int64
	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				return written, fmt.Errorf("write downloaded data: %w", writeErr)
			}
			written += int64(n)
			if progress != nil {
				progress(written, total)
			}
		}
		if readErr == io.EOF {
			return written, nil
		}
		if readErr != nil {
			return written, fmt.Errorf("read downloaded data: %w", readErr)
		}
	}
}

// verifyDownload checks the downloaded bytes against the published SHA256 or,
// when none is available, against the size GitHub reports for the asset.
func verifyDownload(asset *Asset, expectedSum, gotSum string, written int64) error {
	if expectedSum != "" {
		if !strings.EqualFold(expectedSum, gotSum) {
			return fmt.Errorf("контрольная сумма SHA256 не совпала: ожидалась %s, получена %s", expectedSum, gotSum)
		}
		return nil
	}

	if asset.Size > 0 && written != asset.Size {
		return fmt.Errorf("размер загруженного файла %d байт, ожидалось %d", written, asset.Size)
	}
	if written == 0 {
		return fmt.Errorf("загружен пустой файл")
	}
	return nil
}

// lookupChecksum returns the published SHA256 for the named asset, or an empty
// string when the release carries no usable checksums.txt. A missing or
// unreadable checksums file is not fatal — the caller falls back to the size
// check — so problems are logged as warnings only.
func lookupChecksum(ctx context.Context, rel *Release, assetName string) string {
	sumsAsset := rel.AssetByName(checksumsAssetName)
	if sumsAsset == nil {
		log.Printf("[WARN] Release %s has no %s, falling back to size verification", rel.TagName, checksumsAssetName)
		return ""
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sumsAsset.DownloadURL, nil)
	if err != nil {
		log.Printf("[WARN] Checksums request not built: %v", err)
		return ""
	}
	req.Header.Set("User-Agent", userAgent())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[WARN] Checksums not downloaded: %v", err)
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[WARN] Checksums download returned %s", resp.Status)
		return ""
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxChecksumsBytes))
	if err != nil {
		log.Printf("[WARN] Checksums not read: %v", err)
		return ""
	}

	sum := parseChecksums(data)[assetName]
	if sum == "" {
		log.Printf("[WARN] %s has no entry for %s, falling back to size verification", checksumsAssetName, assetName)
	}
	return sum
}

// parseChecksums decodes sha256sum(1) output: one "<hex>␠␠<filename>" line per
// file, where the separator may also be "␠*" for binary mode. Any directory
// prefix on the file name is dropped and unparsable lines are skipped.
func parseChecksums(data []byte) map[string]string {
	sums := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		name = name[strings.LastIndexAny(name, "/\\")+1:]
		if name == "" {
			continue
		}
		sums[name] = fields[0]
	}
	return sums
}
