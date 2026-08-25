package updater

import (
	"context"
	"log"
	"time"
)

// checkInterval is how often the terminal asks GitHub for the latest release.
// Once a day is frequent enough to notice a release the day it lands and rare
// enough to stay far below the unauthenticated API rate limit.
const checkInterval = 24 * time.Hour

// nowFunc is the clock used by the scheduler. It is a package variable so
// tests can pin time down instead of waiting for it.
var nowFunc = time.Now

// ShouldCheck reports whether an update check is due: either none has ever
// run (a zero LastCheck) or the last one was at least checkInterval ago.
//
// A LastCheck in the future (a clock that moved backwards, a hand-edited
// cache) simply postpones the next check; it never causes a check storm.
func ShouldCheck(state State, now time.Time) bool {
	if state.LastCheck.IsZero() {
		return true
	}
	return now.Sub(state.LastCheck) >= checkInterval
}

// checker owns one background check loop. It remembers which version it has
// already announced so a version found today is not re-announced tomorrow.
type checker struct {
	current      string
	onNewVersion func(latest string)
	notified     string
}

// Run watches for new releases in the background until ctx is cancelled.
//
// It returns immediately — performing no request and writing no file — unless
// current is a release version: development builds are never nagged about
// updates. Otherwise it checks straight away when a check is due, then once
// per checkInterval, saving each successful result to the update cache.
//
// onNewVersion is invoked with the release tag when a newer version is found,
// at most once per version for the lifetime of the loop. It is called from the
// background goroutine, so a UI caller must marshal it onto the event loop
// (tview's QueueUpdateDraw).
//
// Run blocks, and is meant to be started with `go updater.Run(...)`. Failures
// are logged as [WARN] and never propagate: an update check must not be able
// to disturb trading.
func Run(ctx context.Context, current string, onNewVersion func(latest string)) {
	if !IsRelease(current) {
		return
	}

	c := &checker{current: current, onNewVersion: onNewVersion}

	for {
		state, err := LoadState()
		if err != nil {
			// No config directory (no home, for example) — the update check
			// silently switches itself off rather than retrying forever.
			log.Printf("[WARN] Update check disabled: %v", err)
			return
		}

		wait := checkInterval
		if ShouldCheck(state, nowFunc()) {
			if err := c.checkOnce(ctx); err != nil {
				log.Printf("[WARN] Update check failed: %v", err)
			}
		} else {
			wait = checkInterval - nowFunc().Sub(state.LastCheck)
		}

		if !sleepCtx(ctx, wait) {
			return
		}
	}
}

// checkOnce performs a single check: fetch the latest release, cache the
// result and announce the version when it is newer than the running one.
//
// The cache is updated only on success, so a failed check does not push the
// next attempt a full day out.
func (c *checker) checkOnce(ctx context.Context) error {
	rel, err := FetchLatestRelease(ctx)
	if err != nil {
		return err
	}

	state := State{
		LastCheck:     nowFunc(),
		LatestVersion: rel.TagName,
		ReleaseURL:    rel.HTMLURL,
		PublishedAt:   rel.PublishedAt,
	}
	if err := SaveState(state); err != nil {
		log.Printf("[WARN] Update state not saved: %v", err)
	}

	if !IsNewer(c.current, rel.TagName) || c.notified == rel.TagName {
		return nil
	}
	c.notified = rel.TagName
	if c.onNewVersion != nil {
		c.onNewVersion(rel.TagName)
	}
	return nil
}

// sleepCtx waits for d, reporting false when ctx was cancelled first. A
// non-positive duration means "due now" and returns immediately.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
