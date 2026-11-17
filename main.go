package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type healthCheck struct {
	url  string
	http int
	time time.Duration
	err  error
}

type checkOperation interface {
	check(url string) healthCheck
}

type httpCheck struct {
	client *http.Client
}

func newHTTPCheck(timeout time.Duration) *httpCheck {
	return &httpCheck{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (h *httpCheck) check(url string) healthCheck {
	start := time.Now()
	resp, err := h.client.Get(url)
	duration := time.Since(start)

	if err != nil {
		return healthCheck{
			url:  url,
			time: duration,
			err:  err,
		}
	}
	defer resp.Body.Close()

	return healthCheck{
		url:  url,
		http: resp.StatusCode,
		time: duration,
		err:  nil,
	}
}

type stats struct {
	mu          sync.Mutex
	totalChecks int
	success200  int
	failed      int
}

func (s *stats) update(res healthCheck) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.totalChecks++
	if res.err != nil {
		s.failed++
		return
	}
	if res.http == http.StatusOK {
		s.success200++
	}
}

func main() {
	var checker checkOperation = newHTTPCheck(5 * time.Second)

	urls := []string{
		"https://go.dev",
		"https://github.com",
		"https://stackoverflow.com",
		"https://ooooo.com",
		"https://invalid-fake-host-test.org",
	}

	var wg sync.WaitGroup
	report := &stats{}

	totalStart := time.Now()

	for _, u := range urls {
		wg.Add(1)

		go func(targetURL string) {
			defer wg.Done()

			res := checker.check(targetURL)

			report.update(res)

			if res.err != nil {
				fmt.Printf("[FAIL] %-40s | Час: %-10v | Помилка: %v\n", res.url, res.time, res.err)
				return
			}
			fmt.Printf("[OK]   %-40s | Час: %-10v | Код: %d\n", res.url, res.time, res.http)
		}(u)
	}

	wg.Wait()

	fmt.Printf("\nЗагальний час виконання: %v\n", time.Since(totalStart))

	fmt.Println("\n--- ПІДСУМКОВА СТАТИСТИКА ---")
	fmt.Printf("Всього перевірено адрес : %d\n", report.totalChecks)
	fmt.Printf("Успішних (HTTP 200)      : %d\n", report.success200)
	fmt.Printf("Помилок з'єднання        : %d\n", report.failed)
}
