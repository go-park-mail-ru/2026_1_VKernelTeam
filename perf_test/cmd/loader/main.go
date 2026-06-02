// Package main — нагрузочный инструмент для catalog-сервиса.
//
// Два режима:
//   create — POST /ads, multipart с JSON в поле data (без файлов)
//   read   — GET  /ads/{id}, id берётся случайно из [1..max-id]
//
// Отчёт: total, errors, RPS, latency p50/p95/p99/avg/max.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	mathrand "math/rand"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: loader <create|read|register> [flags]")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "register":
		cmdRegister(os.Args[2:])
	case "create":
		cmdCreate(os.Args[2:])
	case "read":
		cmdRead(os.Args[2:])
	default:
		fmt.Println("unknown subcommand:", os.Args[1])
		os.Exit(2)
	}
}

type session struct {
	token     string
	csrfToken string
	userID    int64
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func register(base, email, password, name string) (*session, error) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 10 * time.Second}

	body, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
		"name":     name,
	})
	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("register status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var sess session
	for _, c := range resp.Cookies() {
		switch c.Name {
		case "token":
			sess.token = c.Value
		case "csrf_token":
			sess.csrfToken = c.Value
		}
	}
	var ru struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(respBody, &ru)
	sess.userID = ru.ID
	if sess.token == "" {
		return nil, fmt.Errorf("no token cookie set: %s", string(respBody))
	}
	return &sess, nil
}

func login(base, email, password string) (*session, error) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 10 * time.Second}
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("login status=%d body=%s", resp.StatusCode, string(respBody))
	}
	var sess session
	for _, c := range resp.Cookies() {
		switch c.Name {
		case "token":
			sess.token = c.Value
		case "csrf_token":
			sess.csrfToken = c.Value
		}
	}
	return &sess, nil
}

func cmdRegister(args []string) {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	base := fs.String("base", "http://localhost:8001", "auth base URL")
	email := fs.String("email", "", "email (empty = random)")
	password := fs.String("password", "Password1!", "password")
	name := fs.String("name", "PerfTester", "name")
	_ = fs.Parse(args)

	if *email == "" {
		*email = fmt.Sprintf("perf_%s@test.local", randHex(6))
	}
	sess, err := register(*base, *email, *password, *name)
	if err != nil {
		log.Fatalf("register failed: %v", err)
	}
	out := map[string]any{
		"email":      *email,
		"password":   *password,
		"token":      sess.token,
		"csrf_token": sess.csrfToken,
		"user_id":    sess.userID,
	}
	enc, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(enc))
}

// CreateAdRequest mirror of services/catalog/internal/domain/dto.CreateAdRequest minus server-only fields.
type CreateAdRequest struct {
	CategoryID  int64   `json:"category_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Price       int64   `json:"price"`
	Status      string  `json:"status"`
	Location    string  `json:"location"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
}

func buildMultipart(req CreateAdRequest) ([]byte, string) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	data, _ := json.Marshal(req)
	_ = mw.WriteField("data", string(data))
	mw.Close()
	return buf.Bytes(), mw.FormDataContentType()
}

func genCreateRequest(rng *mathrand.Rand, categories []int64) CreateAdRequest {
	cat := categories[rng.Intn(len(categories))]
	title := fmt.Sprintf("Объявление №%s", randHex(6))     // 5..150 chars
	desc := fmt.Sprintf("Описание товара. Случайные данные %s. Тест нагрузки PostgreSQL.", randHex(16))
	return CreateAdRequest{
		CategoryID:  cat,
		Title:       title,
		Description: desc,
		Price:       int64(rng.Intn(100_000) + 100),
		Status:      "active",
		Location:    "Москва",
		Lat:         55.75 + rng.Float64()*0.1,
		Lon:         37.61 + rng.Float64()*0.1,
	}
}

type metric struct {
	dur time.Duration
	ok  bool
}

func runWorkers(total, concurrency int, fn func(workerID, iter int) bool, timeoutPerReq time.Duration) report {
	jobs := make(chan int, total)
	results := make(chan metric, total)
	var wg sync.WaitGroup
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(wid int) {
			defer wg.Done()
			for it := range jobs {
				start := time.Now()
				ok := fn(wid, it)
				results <- metric{dur: time.Since(start), ok: ok}
			}
		}(w)
	}

	startAll := time.Now()
	for i := 0; i < total; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	close(results)
	wall := time.Since(startAll)

	durs := make([]time.Duration, 0, total)
	var okCount, errCount int64
	for m := range results {
		durs = append(durs, m.dur)
		if m.ok {
			okCount++
		} else {
			errCount++
		}
	}
	return makeReport(durs, okCount, errCount, wall)
}

type report struct {
	Total       int           `json:"total"`
	OK          int64         `json:"ok"`
	Errors      int64         `json:"errors"`
	Wall        time.Duration `json:"wall"`
	RPS         float64       `json:"rps"`
	AvgLatency  time.Duration `json:"avg_latency"`
	P50         time.Duration `json:"p50"`
	P95         time.Duration `json:"p95"`
	P99         time.Duration `json:"p99"`
	MaxLatency  time.Duration `json:"max_latency"`
}

func makeReport(durs []time.Duration, ok, errs int64, wall time.Duration) report {
	sort.Slice(durs, func(i, j int) bool { return durs[i] < durs[j] })
	var total time.Duration
	for _, d := range durs {
		total += d
	}
	pct := func(p float64) time.Duration {
		if len(durs) == 0 {
			return 0
		}
		idx := int(float64(len(durs)-1) * p)
		return durs[idx]
	}
	rep := report{
		Total: len(durs), OK: ok, Errors: errs, Wall: wall,
		P50: pct(0.50), P95: pct(0.95), P99: pct(0.99),
	}
	if len(durs) > 0 {
		rep.AvgLatency = total / time.Duration(len(durs))
		rep.MaxLatency = durs[len(durs)-1]
	}
	if wall > 0 {
		rep.RPS = float64(len(durs)) / wall.Seconds()
	}
	return rep
}

func printReport(name string, r report) {
	fmt.Printf("\n== %s ==\n", name)
	fmt.Printf("total:    %d\n", r.Total)
	fmt.Printf("ok:       %d\n", r.OK)
	fmt.Printf("errors:   %d\n", r.Errors)
	fmt.Printf("wall:     %s\n", r.Wall)
	fmt.Printf("RPS:      %.2f\n", r.RPS)
	fmt.Printf("avg:      %s\n", r.AvgLatency)
	fmt.Printf("p50:      %s\n", r.P50)
	fmt.Printf("p95:      %s\n", r.P95)
	fmt.Printf("p99:      %s\n", r.P99)
	fmt.Printf("max:      %s\n", r.MaxLatency)
}

func saveReportJSON(path, name string, r report) {
	if path == "" {
		return
	}
	out := struct {
		Name string `json:"name"`
		report
	}{Name: name, report: r}
	data, _ := json.MarshalIndent(out, "", "  ")
	_ = os.WriteFile(path, data, 0o644)
}

func cmdCreate(args []string) {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	authBase := fs.String("auth-base", "http://localhost:8001", "auth service base URL")
	catalogBase := fs.String("catalog-base", "http://localhost:8004", "catalog service base URL")
	email := fs.String("email", "", "existing user email (empty = register new)")
	password := fs.String("password", "Password1!", "password")
	name := fs.String("name", "PerfTester", "name")
	categoriesCSV := fs.String("categories", "1,2,3,4,5", "comma-separated category IDs to pick from")
	total := fs.Int("total", 100_000, "total requests")
	concurrency := fs.Int("c", 50, "concurrency (workers)")
	out := fs.String("out", "", "write JSON report to file")
	_ = fs.Parse(args)

	// auth
	var sess *session
	var err error
	if *email == "" {
		*email = fmt.Sprintf("perf_%s@test.local", randHex(6))
		log.Printf("registering new user %s", *email)
		sess, err = register(*authBase, *email, *password, *name)
	} else {
		log.Printf("logging in as %s", *email)
		sess, err = login(*authBase, *email, *password)
	}
	if err != nil {
		log.Fatalf("auth failed: %v", err)
	}
	log.Printf("auth ok, token len=%d csrf len=%d", len(sess.token), len(sess.csrfToken))

	categories := parseInt64CSV(*categoriesCSV)
	if len(categories) == 0 {
		log.Fatalf("no categories provided")
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        *concurrency * 2,
			MaxIdleConnsPerHost: *concurrency * 2,
			MaxConnsPerHost:     *concurrency * 2,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	var ok2xx, err5xx, err4xx atomic.Int64
	fn := func(_, iter int) bool {
		rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano() + int64(iter)))
		req := genCreateRequest(rng, categories)
		body, ct := buildMultipart(req)
		httpReq, _ := http.NewRequest(http.MethodPost, *catalogBase+"/api/v1/ads", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", ct)
		httpReq.Header.Set("X-CSRF-Token", sess.csrfToken)
		httpReq.AddCookie(&http.Cookie{Name: "token", Value: sess.token})
		httpReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: sess.csrfToken})
		resp, err := httpClient.Do(httpReq)
		if err != nil {
			err5xx.Add(1)
			return false
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 500 {
			err5xx.Add(1)
			return false
		}
		if resp.StatusCode >= 400 {
			err4xx.Add(1)
			return false
		}
		ok2xx.Add(1)
		return true
	}

	log.Printf("starting create: total=%d c=%d", *total, *concurrency)
	rep := runWorkers(*total, *concurrency, fn, 30*time.Second)
	log.Printf("breakdown: 2xx=%d 4xx=%d 5xx/net=%d", ok2xx.Load(), err4xx.Load(), err5xx.Load())
	printReport("CREATE /ads", rep)
	saveReportJSON(*out, "CREATE /ads", rep)
}

func cmdRead(args []string) {
	fs := flag.NewFlagSet("read", flag.ExitOnError)
	catalogBase := fs.String("catalog-base", "http://localhost:8004", "catalog service base URL")
	mode := fs.String("mode", "by-id", "by-id | list | search")
	minID := fs.Int64("min-id", 1, "min ad id (inclusive)")
	maxID := fs.Int64("max-id", 100_000, "max ad id (inclusive)")
	searchQ := fs.String("q", "Объявление", "search query")
	total := fs.Int("total", 10_000, "total requests")
	concurrency := fs.Int("c", 50, "concurrency (workers)")
	out := fs.String("out", "", "write JSON report to file")
	_ = fs.Parse(args)

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        *concurrency * 2,
			MaxIdleConnsPerHost: *concurrency * 2,
			MaxConnsPerHost:     *concurrency * 2,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	var ok2xx, err4xx, err5xx atomic.Int64
	fn := func(_, iter int) bool {
		rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano() + int64(iter)))
		var url string
		switch *mode {
		case "by-id":
			id := *minID + rng.Int63n(*maxID-*minID+1)
			url = fmt.Sprintf("%s/api/v1/ads/%d", *catalogBase, id)
		case "list":
			url = *catalogBase + "/api/v1/ads"
		case "search":
			url = fmt.Sprintf("%s/api/v1/ads/search?query=%s", *catalogBase, *searchQ)
		default:
			log.Fatalf("unknown mode: %s", *mode)
		}
		resp, err := httpClient.Get(url)
		if err != nil {
			err5xx.Add(1)
			return false
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 500 {
			err5xx.Add(1)
			return false
		}
		if resp.StatusCode >= 400 {
			err4xx.Add(1)
			return false
		}
		ok2xx.Add(1)
		return true
	}

	log.Printf("starting read mode=%s total=%d c=%d", *mode, *total, *concurrency)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = ctx
	rep := runWorkers(*total, *concurrency, fn, 30*time.Second)
	log.Printf("breakdown: 2xx=%d 4xx=%d 5xx/net=%d", ok2xx.Load(), err4xx.Load(), err5xx.Load())
	printReport("READ "+*mode, rep)
	saveReportJSON(*out, "READ "+*mode, rep)
}

func parseInt64CSV(s string) []int64 {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var n int64
		_, err := fmt.Sscan(p, &n)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return out
}
