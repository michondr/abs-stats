package main

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	sessionsPerPage = 100
	metaBackfill    = "backfill_done"
)

// Syncer pulls listening sessions + covers from Audiobookshelf into the store.
// Syncs are collapsed by singleflight so concurrent page loads trigger at most
// one in-flight sync.
type Syncer struct {
	client    *Client
	store     *Store
	coversDir string
	loc       *time.Location
	st        *status
	sf        singleflight.Group
}

func newSyncer(client *Client, store *Store, coversDir string, loc *time.Location, st *status) *Syncer {
	return &Syncer{client: client, store: store, coversDir: coversDir, loc: loc, st: st}
}

// trigger kicks off a background sync (deduped). It never blocks the caller.
func (sy *Syncer) trigger() {
	go sy.sf.Do("sync", func() (any, error) {
		sy.run()
		return nil, nil
	})
}

func (sy *Syncer) run() {
	first := !sy.st.snapshot().Ready
	sy.st.set(func(s *status) {
		s.Building = true
		s.Error = ""
		if first {
			s.Message = "Connecting to Audiobookshelf…"
		}
	})
	log.Printf("sync: starting (initial=%v)", first)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := sy.sync(ctx, first); err != nil {
		log.Printf("sync: failed: %v", err)
		sy.st.set(func(s *status) {
			s.Building = false
			if first {
				s.Error = "Could not sync with Audiobookshelf: " + err.Error()
			}
		})
		return
	}

	n, _ := sy.store.sessionCount()
	log.Printf("sync: done (%d sessions stored)", n)
	sy.st.set(func(s *status) {
		s.Ready = true
		s.Building = false
		s.Error = ""
		s.Message = ""
		s.UpdatedAt = time.Now().In(sy.loc).Format(time.RFC3339)
	})
}

// sync pages listening-sessions newest-first, upserting each page. Once the
// initial backfill is complete it stops early at the first page that brings no
// new or changed rows (the already-synced region); during backfill it walks to
// the oldest page, then records backfill_done. Finally it fetches any covers
// not yet on disk.
func (sy *Syncer) sync(ctx context.Context, first bool) error {
	backfillDone := sy.store.getMetaBool(metaBackfill)
	seen := 0
	for page := 0; ; page++ {
		sr, err := sy.client.fetchSessionsPage(ctx, page, sessionsPerPage)
		if err != nil {
			return err
		}
		changed, err := sy.store.upsertSessions(sr.Sessions)
		if err != nil {
			return err
		}
		seen += len(sr.Sessions)
		sy.st.set(func(s *status) {
			s.Fetched = seen
			if first {
				s.Message = "Fetching listening history… " + strconv.Itoa(seen) + " sessions"
			}
		})

		last := len(sr.Sessions) == 0 || page+1 >= sr.NumPages
		if last {
			if !backfillDone {
				if err := sy.store.setMeta(metaBackfill, "1"); err != nil {
					return err
				}
			}
			break
		}
		if backfillDone && changed == 0 {
			break // reached already-synced data; nothing older changed
		}
	}

	return sy.syncCovers(ctx, first)
}

// syncCovers downloads + downscales any book cover not already on disk.
func (sy *Syncer) syncCovers(ctx context.Context, first bool) error {
	ids, err := sy.store.distinctBookIDs()
	if err != nil {
		return err
	}
	var missing []string
	for _, id := range ids {
		if !coverResolved(sy.coversDir, id) { // skip ones we have or have given up on
			missing = append(missing, id)
		}
	}
	for i, id := range missing {
		if err := ctx.Err(); err != nil {
			return err
		}
		if first {
			n := i + 1
			sy.st.set(func(s *status) {
				s.Message = "Fetching covers… " + strconv.Itoa(n) + "/" + strconv.Itoa(len(missing))
			})
		}
		switch err := sy.client.fetchCover(ctx, sy.coversDir, id); {
		case err == nil:
		case errors.Is(err, errCoverGone):
			// deleted from the library — record a miss so we stop refetching.
			markCoverMiss(sy.coversDir, id)
		default:
			log.Printf("sync: cover %s: %v", id, err) // transient; retry next sync
		}
	}
	return nil
}
