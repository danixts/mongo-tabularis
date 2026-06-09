package mongodb

import "sync"

const schemaConcurrency = 8

func runBounded(count, limit int, worker func(index int)) {
	if count == 0 {
		return
	}
	if limit < 1 {
		limit = 1
	}
	slots := make(chan struct{}, limit)
	var wait sync.WaitGroup
	for i := range count {
		wait.Add(1)
		slots <- struct{}{}
		go func(index int) {
			defer wait.Done()
			defer func() { <-slots }()
			worker(index)
		}(i)
	}
	wait.Wait()
}
