package main

import (
	"fmt"
	"sync"
)

type SafeCond struct {
	urls map[string]bool
	mux sync.Mutex
}

type Fetcher interface {
	// Fetch returns the body of URL and
	// a slice of URLs found on that page.
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl uses fetcher to recursively crawl
// pages starting with url, to a maximum of depth.
func Crawl(url string, depth int, fetcher Fetcher, sf *SafeCond, wg *sync.WaitGroup) {
	defer wg.Done()
	if depth <= 0 {
		return
	}
	sf.mux.Lock()
	_, ok := sf.urls[url]
	sf.mux.Unlock()
	if !ok {
		sf.mux.Lock()
		sf.urls[url] = true
		body, urls, err := fetcher.Fetch(url)
		sf.mux.Unlock()
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		data := fmt.Sprintf("found: %s %q\n", url, body)
		fmt.Println(data)
		for _, u := range urls {
			wg.Add(1)
			go Crawl(u, depth-1, fetcher, sf, wg)
		}
	}
	return
}

func main() {
	var wg sync.WaitGroup
	uniUrls := SafeCond{urls: make(map[string]bool)}
	wg.Add(1)
	go Crawl("https://golang.org/", 4, fetcher, &uniUrls, &wg)
	fmt.Println(wg)
	wg.Wait()

}

// fakeFetcher is Fetcher that returns canned results.
type fakeFetcher map[string]*fakeResult

type fakeResult struct {
	body string
	urls []string
}

func (f fakeFetcher) Fetch(url string) (string, []string, error) {
	if res, ok := f[url]; ok {
		return res.body, res.urls, nil
	}
	return "", nil, fmt.Errorf("not found: %s", url)
}

// fetcher is a populated fakeFetcher.
var fetcher = fakeFetcher{
	"https://golang.org/": &fakeResult{
		"The Go Programming Language",
		[]string{
			"https://golang.org/pkg/",
			"https://golang.org/cmd/",
		},
	},
	"https://golang.org/pkg/": &fakeResult{
		"Packages",
		[]string{
			"https://golang.org/",
			"https://golang.org/cmd/",
			"https://golang.org/pkg/fmt/",
			"https://golang.org/pkg/os/",
		},
	},
	"https://golang.org/pkg/fmt/": &fakeResult{
		"Package fmt",
		[]string{
			"https://golang.org/",
			"https://golang.org/pkg/",
		},
	},
	"https://golang.org/pkg/os/": &fakeResult{
		"Package os",
		[]string{
			"https://golang.org/",
			"https://golang.org/pkg/",
		},
	},
}
