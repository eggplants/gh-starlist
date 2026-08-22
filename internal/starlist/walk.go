package starlist

import (
	"fmt"
	"sync"
)

// DefaultWorkers is how many star lists are read at once. Every list costs at
// least one round trip, and they are independent, so reading them one after
// another wastes most of the wall clock on waiting. A handful of concurrent
// GraphQL queries is well within what GitHub serves; going much wider only
// courts the secondary rate limit.
const DefaultWorkers = 8

// ListReposAll reads the repositories of every list, several lists at a time,
// and returns them in the order of lists.
//
// progress, when not nil, is called with the repositories fetched so far
// across every list and the total the lists hold. It is called from several
// goroutines, one at a time.
func (c *Client) ListReposAll(lists []List, workers int, progress func(fetched, total int)) ([][]Repo, error) {
	if workers <= 0 {
		workers = DefaultWorkers
	}
	total := 0
	for _, list := range lists {
		total += list.RepoCount
	}

	var (
		mutex  sync.Mutex
		counts = make([]int, len(lists))
		repos  = make([][]Repo, len(lists))
		errs   = make([]error, len(lists))
		wait   sync.WaitGroup
		slots  = make(chan struct{}, workers)
	)

	for index, list := range lists {
		wait.Add(1)
		go func() {
			defer wait.Done()
			slots <- struct{}{}
			defer func() { <-slots }()

			// Each list reports through its own hook, so the pages of one
			// cannot be mistaken for the count of another; the shared mutex
			// then serializes the sum and the report built from it.
			client := c.withProgress(nil)
			if progress != nil {
				client = c.withProgress(func(fetched, _ int) {
					mutex.Lock()
					defer mutex.Unlock()
					counts[index] = fetched
					sum := 0
					for _, count := range counts {
						sum += count
					}
					progress(sum, total)
				})
			}

			items, err := client.ListRepos(list.ID, 0)
			if err != nil {
				errs[index] = fmt.Errorf("reading list %q: %w", list.Slug, err)
				return
			}
			repos[index] = items
		}()
	}
	wait.Wait()

	// Report the first failure in list order: which list loses the race must
	// not decide what the user is told.
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return repos, nil
}
