// Package main contains the main program entry-point
package main

import (
	"fmt"
	"sync"
)

// main is the main program entry-point.
func main() {
	channel := make(chan string)

	go Crawl("http://golang.org/", 4, fetcher, channel)

	for {
		visitedURL, ok := <- channel
		if !ok {
			break
		}

		fmt.Println("Visited URL: ", visitedURL)
	}
}

// Crawl uses fetcher to recursively crawl pages starting with url, to a maximum of depth.
// Each page crawled will be sent to the channel; when all pages have been crawled, the channel will be closed.
func Crawl(url string, depth int, fetcher Fetcher, channel chan string) {
	if depth <= 0 {
		close(channel)

		return
	}
	body, urls, err := fetcher.Fetch(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Found: %s %q\n", url, body)
	for _, u := range urls {
		go Crawl(u, depth-1, fetcher, channel)
	}
	return
}

//////////
// Fetcher

// Fetcher represents a facility to fetch data from a URL.
type Fetcher interface {
	// Fetch returns the body of URL and a slice of URLs found on that page.
	Fetch(url string) (body string, urls []string, err error)
}

///////////////
// Fake fetcher

// fakeFetcher is Fetcher that returns canned results.
type fakeFetcher map[string]*fakeResult

// fakeResult is the result of fetching a page using a fakeFetcher.
type fakeResult struct {
	body string
	urls []string
}

func (fetcher fakeFetcher) Fetch(url string) (body string, urls []string, err error) {
	fmt.Printf("\t\tFake-fetch: %v\n", url)

	if res, ok := fetcher[url]; ok {
		return res.body, res.urls, nil
	}

	return "", nil, fmt.Errorf("Not found: %s", url)
}

/////////////////
// Cached fetcher

// cachedFetcher aggregates a fetcher and a result cache.
type cachedFetcher struct {
	innerFetcher Fetcher
	cache        *resultCache
}

// Fetch retrieves the body and discovered URLs for the specified URL (using the cache if possible).
func (fetcher *cachedFetcher) Fetch(url string) (body string, urls []string, err error) {
	return fetcher.cache.getOrAdd(url, fetcher.innerFetcher)
}

////////
// Cache

type resultCache struct {
	results map[string]*fakeResult
	mux     *sync.Mutex
}

// Retrieve a fetch result from the cache, or perform the fetch and add its result to the cache.
func (cache *resultCache) getOrAdd(url string, fetcher Fetcher) (string, []string, error) {
	cache.mux.Lock()
	defer cache.mux.Unlock()

	fmt.Printf("\tCache fetch: %v\n", url)

	result, ok := cache.results[url]
	if ok {
		fmt.Printf("\tCache hit: %v\n", url)
		return result.body, result.urls, nil
	}

	fmt.Printf("\tCache miss: %v\n", url)
	body, urls, err := fetcher.Fetch(url)
	if err == nil {
		cache.results[url] = &fakeResult{body, urls}
	}

	return body, urls, err
}

// fetcher is a populated fakeFetcher.
var fetcher = &cachedFetcher{
	innerFetcher: fakeFetcher{
		"http://golang.org/": &fakeResult{
			"The Go Programming Language",
			[]string{
				"http://golang.org/pkg/",
				"http://golang.org/cmd/",
			},
		},
		"http://golang.org/pkg/": &fakeResult{
			"Packages",
			[]string{
				"http://golang.org/",
				"http://golang.org/cmd/",
				"http://golang.org/pkg/fmt/",
				"http://golang.org/pkg/os/",
			},
		},
		"http://golang.org/pkg/fmt/": &fakeResult{
			"Package fmt",
			[]string{
				"http://golang.org/",
				"http://golang.org/pkg/",
			},
		},
		"http://golang.org/pkg/os/": &fakeResult{
			"Package os",
			[]string{
				"http://golang.org/",
				"http://golang.org/pkg/",
			},
		},
	},
	cache: &resultCache{
		make(map[string]*fakeResult), &sync.Mutex{},
	},
}
