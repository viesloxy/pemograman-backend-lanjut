package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

var (
	baseURL     = "http://localhost:3000/api/v1"
	jwtSecret   string
	jwtIssuer   = "api-students"
	loadtestLog *slog.Logger
)

func main() {
	mode := flag.String("mode", "crowd", "crowd | swap")
	concurrency := flag.Int("n", 60, "jumlah request serentak")
	baseID := flag.Int("base", 1, "id student pertama yang dipakai crowd/swap mode")
	role := flag.String("role", "admin", "role pada token crowd mode")
	printToken := flag.Bool("print-token", false, "cetak token untuk -base dan -role lalu keluar")
	flag.StringVar(&jwtSecret, "secret", os.Getenv("JWT_SECRET"), "JWT_SECRET")
	flag.Parse()

	if *printToken {
		t := makeToken(*baseID, *role)
		fmt.Println(t)
		return
	}

	if jwtSecret == "" || len(jwtSecret) < 32 {
		fmt.Println("JWT_SECRET wajib diisi (minimal 32 karakter)")
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	loadtestLog = logger

	token := makeToken(1, *role)
	client := &http.Client{Timeout: 10 * time.Second}

	start := time.Now()
	var results map[string]int

	switch *mode {
	case "crowd":
		results = fire(*concurrency, func(i int) string {
			return enroll(client, token, 1, *baseID+i)
		})
	case "swap":
		// Tahap 0: bagilah mahasiswa rata ke dua pembimbing, bergiliran.
		// Penugasan dijaga permission supervision:manage, jadi memakai
		// token admin. Kuota kedua pembimbing harus cukup untuk n/2.
		for i := 0; i < *concurrency; i++ {
			sup := 1 + i%2
			enroll(client, token, sup, *baseID+i)
		}
		// Tahap 1: seluruh mahasiswa pindah berlawanan arah serentak.
		// Setiap mahasiswa membawa token sendiri supaya yang diuji
		// adalah pindah kelas oleh banyak orang, bukan satu orang.
		results = fire(*concurrency, func(i int) string {
			from := 1 + i%2
			to := 2 - i%2
			return swapSupervisor(client, makeToken(*baseID+i, "user"), from, to)
		})
	default:
		fmt.Println("mode tidak dikenal:", *mode)
		os.Exit(1)
	}

	elapsed := time.Since(start)

	fmt.Printf("\nhasil request:\n")
	for status, count := range results {
		fmt.Printf("  %-24s %d\n", status, count)
	}
	fmt.Printf("waktu total : %v\n", elapsed)

	_ = token
	_ = baseURL
	_ = strings.TrimSpace
	_ = json.Marshal
	_ = io.Discard
}

func makeToken(userID int, role string) string {
	claims := jwt.MapClaims{
		"sub":  fmt.Sprintf("%d", userID),
		"role": role,
		"iss":  jwtIssuer,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(15 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(jwtSecret))
	return signed
}

func enroll(client *http.Client, token string, supervisorID, studentID int) string {
	body := fmt.Sprintf(`{"student_id":%d}`, studentID)
	req, _ := http.NewRequest("POST",
		fmt.Sprintf("%s/supervisors/%d/assign", baseURL, supervisorID),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return "NETWORK_ERROR"
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
}

func swapSupervisor(client *http.Client, token string, from, to int) string {
	body := fmt.Sprintf(`{"from_supervisor_id":%d,"to_supervisor_id":%d}`, from, to)
	req, _ := http.NewRequest("POST",
		fmt.Sprintf("%s/enrollments/switch", baseURL),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return "NETWORK_ERROR"
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode == fiber.StatusOK {
		return "200 OK"
	}
	return fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
}

func fire(n int, job func(i int) string) map[string]int {
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan string, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int, out chan string) {
			defer wg.Done()
			<-start
			out <- job(i)
		}(i, results)
	}

	close(start)
	wg.Wait()
	close(results)

	tally := map[string]int{}
	for r := range results {
		tally[r]++
	}
	return tally
}
