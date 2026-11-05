package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Alat uji beban Modul 9.
//
// Folder tools/ BUKAN bagian aplikasi: ia hanya alat untuk membuktikan ada
// atau tidaknya race condition pada endpoint kelas/KRS. Alat ini membuat
// sendiri access token untuk akun uji mhs001.. karena ia memegang
// JWT_SECRET. Itulah sebabnya JWT_SECRET tidak boleh bocor: siapa pun yang
// memegangnya dapat mencetak token atas nama siapa saja.
var (
	baseURL   = "http://localhost:3000/api/v1"
	jwtSecret string
	jwtIssuer = "praktikum-backend"

	// id kelas sesuai migrations/005_courses.sql.
	courseRebutan = 13 // IF399 "Kelas Rebutan", kuota 5
	courseFrom    = 1  // IF101
	courseTo      = 2  // IF102
)

func main() {
	mode := flag.String("mode", "crowd", "crowd | credits | swap")
	concurrency := flag.Int("n", 60, "jumlah request/pekerja serentak")
	baseID := flag.Int("base", 2, "id user mhs pertama yang dipakai (mhs001)")
	role := flag.String("role", "user", "role pada token yang dicetak alat ini")
	url := flag.String("url", baseURL, "basis URL API")
	printToken := flag.Bool("print-token", false, "cetak token untuk -base dan -role lalu keluar")
	flag.StringVar(&jwtSecret, "secret", os.Getenv("JWT_SECRET"), "JWT_SECRET")
	flag.StringVar(&jwtIssuer, "issuer", "praktikum-backend", "JWT_ISSUER")
	flag.Parse()

	baseURL = *url

	if *printToken {
		fmt.Println(makeToken(*baseID, *role))
		return
	}

	if jwtSecret == "" || len(jwtSecret) < 32 {
		fmt.Println("JWT_SECRET wajib diisi (minimal 32 karakter)")
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	client := &http.Client{Timeout: 15 * time.Second}

	start := time.Now()
	var results map[string]int

	switch *mode {
	case "crowd":
		// 60 mahasiswa (base..base+59) berebut SATU kelas rebutan
		// (IF399, kuota 5). Setiap mahasiswa membawa token sendiri,
		// karena endpoint enroll tidak butuh permission: ia hanya
		// mendaftarkan DIRINYA di kelas tersebut.
		//
		// Hasil yang benar: tepat 5 berhasil, sisanya COURSE_FULL,
		// KONSISTEN tiga kali berturut-turut tanpa reset.
		results = fire(*concurrency, func(i int) string {
			return enroll(client, makeToken(*baseID+i, "user"), courseRebutan)
		})

	case "credits":
		// Sepuluh mahasiswa penyerang, masing-masing berebut 12 kelas
		// (3 SKS) SECARA BERSAMAAN. Batas 24 SKS = 8 kelas.
		//
		// Satu penyerang kadang kebetulan lolos; sepuluh penyerang
		// membuat race condition jauh lebih mudah terlihat. Hasil
		// benar: tidak satu mahasiswa pun melewati 24 SKS.
		const coursesPerStudent = 12
		results = fire(*concurrency*coursesPerStudent, func(k int) string {
			student := *baseID + k/coursesPerStudent
			courseID := 1 + k%coursesPerStudent
			return enroll(client, makeToken(student, roleFor(*role, "user")), courseID)
		})

	case "swap":
		// 20 pasang mahasiswa pindah kelas berlawanan arah
		// (IF101 <-> IF102) SECARA BERSAMAAN.
		//
		// Tahap 0: bagilah mahasiswa rata ke dua kelas, bergiliran.
		// Tahap 1: seluruh mahasiswa pindah berlawanan arah serentak.
		// Setiap mahasiswa membawa token sendiri supaya yang diuji
		// adalah pindah kelas oleh banyak orang, bukan satu orang.
		for i := 0; i < *concurrency; i++ {
			enroll(client, makeToken(*baseID+i, "user"), 1+i%2)
		}
		results = fire(*concurrency, func(i int) string {
			from := 1 + i%2
			to := 2 - i%2
			return swap(client, makeToken(*baseID+i, "user"), from, to)
		})

	default:
		fmt.Println("mode tidak dikenal:", *mode)
		os.Exit(1)
	}

	elapsed := time.Since(start)

	fmt.Printf("\nhasil request (%s):\n", *mode)
	for status, count := range results {
		fmt.Printf("  %-24s %d\n", status, count)
	}
	fmt.Printf("waktu total : %v\n", elapsed)
}

// roleFor memaksa mode credits memakai role pada token yang sah; bila
// pemanggil memberi role khusus untuk crowd, ia dipakai apa adanya.
func roleFor(requested, fallback string) string {
	if strings.TrimSpace(requested) == "" {
		return fallback
	}
	return requested
}

// makeToken mencetak access token yang bentuk claims-nya persis sama
// dengan yang dihasilkan helper.JWTManager aplikasi.
func makeToken(userID int, role string) string {
	claims := jwt.MapClaims{
		"sub":      fmt.Sprintf("%d", userID),
		"username": fmt.Sprintf("mhs%03d", userID),
		"role":     role,
		"iss":      jwtIssuer,
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(30 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(jwtSecret))
	return signed
}

// enroll mendaftarkan pemilik token ke satu kelas.
// Body boleh kosong, tetapi middleware RequireJSON menuntut Content-Type.
func enroll(client *http.Client, token string, courseID int) string {
	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/courses/%d/enroll", baseURL, courseID), nil)
	if err != nil {
		return "REQUEST_ERROR"
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	return do(client, req)
}

// swap memindahkan pemilik token dari satu kelas ke kelas lain.
func swap(client *http.Client, token string, from, to int) string {
	body := fmt.Sprintf(`{"from_course_id":%d,"to_course_id":%d}`, from, to)
	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/enrollments/swap", baseURL), strings.NewReader(body))
	if err != nil {
		return "REQUEST_ERROR"
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	return do(client, req)
}

func do(client *http.Client, req *http.Request) string {
	resp, err := client.Do(req)
	if err != nil {
		return "NETWORK_ERROR"
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
}

// fire menjalankan n pekerjaan secara serentak dan menghitung hasilnya.
//
// Setiap goroutine menunggu di depan "gerbang" start yang sama. Gerbang
// baru dibuka setelah SELURUH goroutine siap, sehingga request benar-benar
// berangkat hampir bersamaan --- bukan bergiliran sesuai urutan dibuatnya.
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

	close(start) // buka gerbang
	wg.Wait()    // tunggu SEMUA goroutine selesai
	close(results)

	tally := map[string]int{}
	for r := range results {
		tally[r]++
	}
	return tally
}
