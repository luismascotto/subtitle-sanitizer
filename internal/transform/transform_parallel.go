package transform

import (
	"runtime"
	"sync"

	"github.com/luismascotto/subtitle-sanitizer/internal/model"
)

// applyAllParallel transforms cues concurrently with a GOMAXPROCS-bounded worker pool,
// then assembles results in input order (same observable output as sequential).
func applyAllParallel(doc model.Document, r Rules) (model.Document, []CueChange) {
	n := len(doc.Cues)
	if n == 0 {
		return assembleDocument(doc, nil)
	}

	outcomes := make([]cueOutcome, n)
	workers := max(min(runtime.GOMAXPROCS(0), n), 1)

	jobs := make(chan int, n)
	for i := range n {
		jobs <- i
	}
	close(jobs)

	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			for i := range jobs {
				outcomes[i] = applyCue(doc.Cues[i], doc.Format, r)
			}
		}()
	}
	wg.Wait()

	return assembleDocument(doc, outcomes)
}
