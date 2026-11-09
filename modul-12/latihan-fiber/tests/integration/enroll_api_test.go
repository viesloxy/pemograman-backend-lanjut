//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/app/service"
	"latihan-fiber/config"
	"latihan-fiber/helper"
	"latihan-fiber/route"
)

// api adalah aplikasi lengkap yang memakai PostgreSQL sungguhan. Bedanya
// dengan production hanya satu: database-nya praktikum_backend_test.
type api struct {
	app *fiber.App
	jwt *helper.JWTManager
}

func newAPI(t *testing.T) api {
	t.Helper()

	raw, err := repository.NewRoleRepository(pool).LoadPermissions(context.Background())
	if err != nil {
		t.Fatalf("memuat permission: %v", err)
	}
	perms := helper.NewPermissionSet(raw)
	jwtManager := helper.NewJWTManager("secret-khusus-test-minimal-32-karakter", "test", time.Minute)

	courseService := service.NewCourseService(
		repository.NewCourseRepository(pool),
		repository.NewOutboxRepository(pool),
		repository.NewTxManager(pool),
	)

	app := config.NewApp(slog.New(slog.NewTextHandler(io.Discard, nil)), route.Dependencies{
		Pool:          pool,
		JWT:           jwtManager,
		Permissions:   perms,
		CourseService: courseService,
	})
	return api{app: app, jwt: jwtManager}
}

// enroll mengirim POST /courses/:id/enroll atas nama student dan
// mengembalikan status code beserta field "code" pada response.
func (a api) enroll(t *testing.T, student model.User, courseID int) (int, string) {
	t.Helper()

	token, err := a.jwt.GenerateAccess(model.User{
		ID:       student.ID,
		Username: student.Username,
		Role:     "user",
	})
	if err != nil {
		t.Errorf("membuat token: %v", err)
		return 0, ""
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+strconv.Itoa(courseID)+"/enroll", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.app.Test(req, -1)
	if err != nil {
		// t.Errorf, bukan t.Fatalf: fungsi ini juga dipanggil dari
		// goroutine, dan t.Fatalf hanya boleh dipanggil dari goroutine test itu sendiri.
		t.Errorf("request enroll: %v", err)
		return 0, ""
	}
	defer resp.Body.Close()

	var body struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return resp.StatusCode, body.Code
}

func countRows(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", sql, err)
	}
	return n
}

func TestEnrollEndpoint_WritesOutboxOnlyOnSuccess(t *testing.T) {
	a := newAPI(t)
	course := seedCourse(t, 3, 40)
	student := seedStudent(t)

	if status, _ := a.enroll(t, student, course.ID); status != http.StatusCreated {
		t.Fatalf("pendaftaran pertama: status = %d, seharusnya 201", status)
	}
	if status, code := a.enroll(t, student, course.ID); status != http.StatusConflict || code != "ALREADY_ENROLLED" {
		t.Fatalf("pendaftaran kedua: dapat %d %s, seharusnya 409 ALREADY_ENROLLED", status, code)
	}

	events := countRows(t,
		`SELECT COUNT(*) FROM outbox WHERE (payload->>'user_id')::INT = $1 AND event_type = $2`,
		student.ID, model.EventEnrollmentCreated)
	if events != 1 {
		t.Errorf("event enrollment.created = %d, seharusnya 1 (pendaftaran yang gagal tidak boleh meninggalkan jejak)", events)
	}
}

// Satu mahasiswa mengirim sepuluh pendaftaran SERENTAK ke sepuluh kelas
// 3 SKS. Batasnya 24 SKS, maka tepat delapan yang boleh berhasil.
// Test ini adalah versi otomatis dari uji beban mode credits pada Modul 9.
func TestEnrollEndpoint_ConcurrentCreditLimit(t *testing.T) {
	a := newAPI(t)
	student := seedStudent(t)

	const attempts = 10
	courses := make([]model.Course, attempts)
	for i := range courses {
		courses[i] = seedCourse(t, 3, 40)
	}

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results = map[int]int{}
		start   = make(chan struct{})
	)
	for _, c := range courses {
		wg.Add(1)
		go func(courseID int) {
			defer wg.Done()
			<-start // tunggu aba-aba agar semua request berangkat bersamaan
			status, _ := a.enroll(t, student, courseID)
			mu.Lock()
			results[status]++
			mu.Unlock()
		}(c.ID)
	}
	close(start) // aba-aba: semua goroutine dilepas pada saat yang sama
	wg.Wait()

	maxCourses := service.MaxCreditsPerSemester / 3
	if results[http.StatusCreated] != maxCourses {
		t.Errorf("berhasil = %d, seharusnya %d (hasil lengkap: %v)", results[http.StatusCreated], maxCourses, results)
	}
	enrolled := countRows(t, `SELECT COUNT(*) FROM enrollments WHERE user_id = $1`, student.ID)
	if enrolled != maxCourses {
		t.Errorf("baris enrollments = %d, seharusnya %d", enrolled, maxCourses)
	}
}

// do mengirim request HTTP apa pun (dengan body JSON opsional) ke aplikasi
// atas nama student dan mengembalikan status code beserta field "code"
// pada response. Helper umum untuk endpoint selain enroll.
func (a api) do(t *testing.T, student model.User, method, path, body string) (int, string) {
	t.Helper()

	token, err := a.jwt.GenerateAccess(model.User{
		ID:       student.ID,
		Username: student.Username,
		Role:     "user",
	})
	if err != nil {
		t.Fatalf("membuat token: %v", err)
	}

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.app.Test(req, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	var parsed struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&parsed)
	return resp.StatusCode, parsed.Code
}

// D.2 butir 1 — swap ke kelas yang sudah penuh harus ditolak 409
// COURSE_FULL, dan SELURUH jejaknya dibatalkan: mahasiswa tetap terdaftar
// di kelas asal, kedua enrolled_count tidak berubah, dan tidak ada event
// enrollment.swapped di outbox.
func TestSwapEndpoint_RejectedWhenTargetFull(t *testing.T) {
	a := newAPI(t)
	ctx := context.Background()

	from := seedCourse(t, 3, 40) // kelas asal, masih longgar
	to := seedCourse(t, 3, 1)    // kelas tujuan dengan satu kursi
	student := seedStudent(t)

	if status, _ := a.enroll(t, student, from.ID); status != http.StatusCreated {
		t.Fatalf("pendaftaran awal: status = %d, seharusnya 201", status)
	}

	// Kursi tunggal kelas tujuan diambil lebih dulu, langsung lewat
	// repository — persis skenario "kelas baru penuh".
	repo := repository.NewCourseRepository(pool)
	if _, err := repo.IncrementEnrolled(ctx, to.ID); err != nil {
		t.Fatalf("mengisi kursi kelas tujuan: %v", err)
	}

	status, code := a.do(t, student, http.MethodPost, "/api/v1/enrollments/swap",
		fmt.Sprintf(`{"from_course_id": %d, "to_course_id": %d}`, from.ID, to.ID))
	if status != http.StatusConflict || code != "COURSE_FULL" {
		t.Fatalf("swap: dapat %d %s, seharusnya 409 COURSE_FULL", status, code)
	}

	if rows := countRows(t,
		`SELECT COUNT(*) FROM enrollments WHERE user_id = $1 AND course_id = $2`,
		student.ID, from.ID); rows != 1 {
		t.Errorf("enrollment di kelas asal = %d, seharusnya 1 (mahasiswa tetap di kelas asal)", rows)
	}
	if got := countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, from.ID); got != 1 {
		t.Errorf("enrolled_count kelas asal = %d, seharusnya 1", got)
	}
	if got := countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, to.ID); got != 1 {
		t.Errorf("enrolled_count kelas tujuan = %d, seharusnya 1", got)
	}
	if events := countRows(t,
		`SELECT COUNT(*) FROM outbox WHERE (payload->>'user_id')::INT = $1 AND event_type = $2`,
		student.ID, model.EventEnrollmentSwapped); events != 0 {
		t.Errorf("event enrollment.swapped = %d, seharusnya 0 (swap yang gagal tidak boleh meninggalkan jejak)", events)
	}
}

// D.2 butir 2 — keluar dari kelas menghasilkan 204, baris enrollment
// hilang, enrolled_count berkurang satu, dan outbox berisi TEPAT SATU
// event enrollment.dropped.
func TestDropEndpoint_RemovesEnrollmentAndWritesEvent(t *testing.T) {
	a := newAPI(t)
	course := seedCourse(t, 3, 40)
	student := seedStudent(t)

	if status, _ := a.enroll(t, student, course.ID); status != http.StatusCreated {
		t.Fatalf("pendaftaran: status = %d, seharusnya 201", status)
	}

	status, code := a.do(t, student, http.MethodDelete,
		"/api/v1/courses/"+strconv.Itoa(course.ID)+"/enroll", "")
	if status != http.StatusNoContent || code != "" {
		t.Fatalf("drop: dapat %d %q, seharusnya 204 tanpa body", status, code)
	}

	if rows := countRows(t,
		`SELECT COUNT(*) FROM enrollments WHERE user_id = $1 AND course_id = $2`,
		student.ID, course.ID); rows != 0 {
		t.Errorf("baris enrollment = %d, seharusnya 0 setelah keluar", rows)
	}
	if got := countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, course.ID); got != 0 {
		t.Errorf("enrolled_count = %d, seharusnya 0 setelah keluar", got)
	}
	if events := countRows(t,
		`SELECT COUNT(*) FROM outbox WHERE (payload->>'user_id')::INT = $1 AND event_type = $2`,
		student.ID, model.EventEnrollmentDropped); events != 1 {
		t.Errorf("event enrollment.dropped = %d, seharusnya tepat 1", events)
	}
}