package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/sync/semaphore"
)

var (
	tasks = runtime.GOMAXPROCS(-1) * 16
)

func init() {
	flag.IntVar(&tasks, "tasks", tasks, "number of concurrent tasks to perform")
	flag.Parse()
}

func main() {
	r, err := http.Get(os.Args[1])
	if err != nil {
		log.Fatalln("failed to get URL:", err)
		return
	}

	defer r.Body.Close()

	doc, err := goquery.NewDocumentFromReader(r.Body)
	if err != nil {
		log.Fatalln("failed to parse HTML:", err)
	}
	r.Body.Close()

	wg := sync.WaitGroup{}
	sema := semaphore.NewWeighted(int64(tasks))

	doc.Find("a.fileThumb").Each(func(i int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok {
			return
		}

		sema.Acquire(context.Background(), 1)
		wg.Add(1)

		go func() {
			downloadAndSave(href)
			sema.Release(1)
			wg.Done()
		}()
	})

	wg.Wait()
}

func downloadAndSave(s string) {
	s = "https:" + s

	if _, err := os.Stat(path.Base(s)); err == nil {
		return
	}

	r, err := http.Get(s)
	if err != nil {
		log.Printf("failed to fetch %s: %v\n", s, err)
		return
	}
	defer r.Body.Close()

	f, err := os.Create(path.Base(s))
	if err != nil {
		log.Println("failed to create file:", err)
		return
	}
	defer f.Close()

	if _, err = io.Copy(f, r.Body); err != nil {
		log.Println("failed to download:", err)
	}
}
