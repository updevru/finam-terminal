package api

import (
	"fmt"
	"log"
	"strings"
	"time"

	"finam-terminal/models"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/assets"
)

const (
	// indexCacheTTL bounds how long a fetched index composition is reused. The
	// composition of an index changes a few times a year, so a day-long cache
	// keeps the whole feature at 1-2 GetConstituents calls per session.
	indexCacheTTL = 24 * time.Hour

	// maxConstituentPages bounds the pagination loop, so a server that keeps
	// handing out cursors can never spin it forever.
	maxConstituentPages = 10
)

// indexCacheEntry is one cached index composition together with the time it was
// fetched, which drives the TTL.
type indexCacheEntry struct {
	constituents []models.IndexConstituent
	fetchedAt    time.Time
}

// GetIndexConstituents returns the composition of a stock index
// (AssetsService.GetConstituents), collected across every page the API offers.
//
// The result is cached per index symbol for indexCacheTTL. A refetch that fails
// — or comes back empty — keeps serving the previous composition rather than
// blanking the view (stale-on-error); only a failure with nothing cached is
// reported to the caller, which then shows a message and offers a manual retry.
// Nothing here retries on its own.
//
// The order the API returns is preserved: sorting is a presentation decision.
func (c *Client) GetIndexConstituents(indexSymbol string) ([]models.IndexConstituent, error) {
	if cached, ok := c.cachedIndexConstituents(indexSymbol, true); ok {
		return cached, nil
	}

	fresh, err := c.fetchIndexConstituents(indexSymbol)
	if err != nil {
		if stale, ok := c.cachedIndexConstituents(indexSymbol, false); ok {
			log.Printf("[WARN] Index %s refresh failed, keeping the cached composition: %v", indexSymbol, err)
			return stale, nil
		}
		return nil, err
	}

	c.storeIndexConstituents(indexSymbol, fresh)
	return fresh, nil
}

// fetchIndexConstituents walks the cursor pagination and maps the result. An
// empty composition is reported as an error: the API answering with nothing is
// a failure, not a valid index with no members, and must not reach the cache.
func (c *Client) fetchIndexConstituents(indexSymbol string) ([]models.IndexConstituent, error) {
	ctx, cancel := c.getContext()
	defer cancel()

	var (
		result []models.IndexConstituent
		cursor int64
	)

	for page := 0; page < maxConstituentPages; page++ {
		resp, err := c.assetsClient.GetConstituents(ctx, &assets.GetConstituentsRequest{
			Symbol: indexSymbol,
			Cursor: cursor,
		})
		if err != nil {
			c.logGRPCError("AssetsService", "GetConstituents", err,
				fmt.Sprintf("Symbol: %s | Cursor: %d", indexSymbol, cursor))
			return nil, fmt.Errorf("failed to get constituents of %s: %w", indexSymbol, err)
		}

		for _, item := range resp.Constituents {
			mapped, ok := mapConstituent(item)
			if !ok {
				continue
			}
			result = append(result, mapped)
		}

		cursor = resp.NextCursor
		if cursor == 0 {
			break
		}
		if page == maxConstituentPages-1 {
			log.Printf("[WARN] Index %s pagination stopped at the %d-page guard, using %d constituent(s) collected so far",
				indexSymbol, maxConstituentPages, len(result))
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("index %s returned no constituents", indexSymbol)
	}
	return result, nil
}

// mapConstituent converts one SDK component into the model. A component without
// a full ticker@mic symbol is dropped: it cannot be subscribed to, opened as a
// profile or ordered, so a row for it would be a dead end.
func mapConstituent(item *assets.Constituents) (models.IndexConstituent, bool) {
	if item == nil {
		return models.IndexConstituent{}, false
	}

	ticker, _, hasMic := strings.Cut(item.Symbol, "@")
	if !hasMic || ticker == "" {
		log.Printf("[WARN] Skipping index constituent with an unusable symbol %q", item.Symbol)
		return models.IndexConstituent{}, false
	}

	return models.IndexConstituent{
		Symbol: item.Symbol,
		Ticker: ticker,
		Name:   item.Name,
		Sector: item.Sector,
		Weight: parseDecimalFloat(item.Weight),
	}, true
}

// cachedIndexConstituents returns a copy of the cached composition. With
// respectTTL it only reports an entry that is still fresh; without it, any
// entry qualifies — that is the stale-on-error path.
func (c *Client) cachedIndexConstituents(indexSymbol string, respectTTL bool) ([]models.IndexConstituent, bool) {
	c.indexMu.RLock()
	defer c.indexMu.RUnlock()

	entry, ok := c.indexCache[indexSymbol]
	if !ok || len(entry.constituents) == 0 {
		return nil, false
	}
	if respectTTL && time.Since(entry.fetchedAt) >= indexCacheTTL {
		return nil, false
	}
	return append([]models.IndexConstituent(nil), entry.constituents...), true
}

// storeIndexConstituents caches a freshly fetched composition.
func (c *Client) storeIndexConstituents(indexSymbol string, constituents []models.IndexConstituent) {
	c.indexMu.Lock()
	defer c.indexMu.Unlock()

	if c.indexCache == nil {
		c.indexCache = make(map[string]indexCacheEntry)
	}
	c.indexCache[indexSymbol] = indexCacheEntry{
		constituents: append([]models.IndexConstituent(nil), constituents...),
		fetchedAt:    time.Now(),
	}
}
