package main

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// defaultLimit dan maxLimit menjaga endpoint daftar tetap aman.
// Batas atas 50 dipilih karena satu halaman data mahasiswa yang wajar
// dibaca manusia ada di kisaran itu. Tanpa batas ini, siapa pun bisa
// mengirim ?limit=99999999 dan memaksa server menyusun respons raksasa
// dalam sekali permintaan.
const (
	defaultPage  = 1
	defaultLimit = 10
	maxLimit     = 50
	maxPage      = 100000
)

// allowedSort adalah daftar putih field yang boleh dipakai mengurutkan.
// Daftar putih dipilih, bukan daftar hitam, supaya nama field asing dari
// klien tidak pernah sampai ke lapisan data. Hari ini data masih di
// memori, tapi kebiasaan ini dibentuk sebelum basis data masuk di modul 3.
var allowedSort = map[string]bool{
	"id":         true,
	"nim":        true,
	"name":       true,
	"grade":      true,
	"created_at": true,
}

// ---------- Amplop respons ----------
// Semua respons keluar lewat fungsi-fungsi di bawah supaya bentuknya
// konsisten di seluruh endpoint, termasuk yang gagal.

func ok(c *fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusOK).JSON(WebResponse{
		Success: true, Message: message, Data: data,
	})
}

func okList(c *fiber.Ctx, message string, data any, meta *Meta) error {
	return c.Status(fiber.StatusOK).JSON(WebResponse{
		Success: true, Message: message, Data: data, Meta: meta,
	})
}

// created menaruh header Location sebelum menulis body, supaya klien tahu
// alamat sumber daya yang baru dibuat tanpa harus menebak. Header harus
// dipasang sebelum respons dikirim, tidak bisa sesudahnya.
func created(c *fiber.Ctx, message string, data any, location string) error {
	c.Set("Location", location)
	return c.Status(fiber.StatusCreated).JSON(WebResponse{
		Success: true, Message: message, Data: data,
	})
}

// noContent memakai SendStatus, bukan JSON, karena 204 memang tidak
// boleh membawa body sama sekali.
func noContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

func fail(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(WebResponse{Success: false, Message: message})
}

// failValidation selalu 422 dan selalu menyebut field mana yang bermasalah,
// supaya aplikasi klien bisa menandai kolom yang salah pada formulirnya.
func failValidation(c *fiber.Ctx, errs map[string]string) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(WebResponse{
		Success: false, Message: "validasi gagal", Errors: errs,
	})
}

// ---------- Pembacaan query string ----------

// parseListQuery membaca query string lalu memberi nilai bawaan yang aman.
// Prinsipnya satu: masukan dari klien tidak pernah dipercaya begitu saja.
func parseListQuery(c *fiber.Ctx) ListQuery {
	q := ListQuery{
		Page:   c.QueryInt("page", defaultPage),
		Limit:  c.QueryInt("limit", defaultLimit),
		Search: strings.TrimSpace(c.Query("search")),
		Sort:   strings.ToLower(strings.TrimSpace(c.Query("sort", "id"))),
		Order:  strings.ToLower(strings.TrimSpace(c.Query("order", "asc"))),
	}

	if q.Page < 1 {
		q.Page = defaultPage
	}
	if q.Page > maxPage { // hindari (page-1)*limit yang meleset dan bikin slice panic
		q.Page = maxPage
	}
	if q.Limit < 1 {
		q.Limit = defaultLimit
	}
	if q.Limit > maxLimit { // batas atas wajib ada
		q.Limit = maxLimit
	}
	if !allowedSort[q.Sort] { // daftar putih: selain yang tercantum kembali ke id
		q.Sort = "id"
	}
	if q.Order != "desc" {
		q.Order = "asc"
	}

	if raw := c.Query("is_active"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			q.IsActive = &v
		}
	}
	if raw := c.Query("min_grade"); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			q.MinGrade = &v
		}
	}
	if raw := c.Query("max_grade"); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			q.MaxGrade = &v
		}
	}

	return q
}

// ---------- Validasi ----------

// nimValid memastikan NIM hanya berisi angka dengan panjang wajar.
// NIM Universitas Airlangga sepanjang 15 digit termasuk di dalam rentang ini.
func nimValid(nim string) bool {
	if len(nim) < 9 || len(nim) > 15 {
		return false
	}
	for _, r := range nim {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// validasiIsi memeriksa isi data mahasiswa dan mengumpulkan pesan per
// field. Fungsi ini dipakai bersama POST, PUT, dan PATCH: parameter
// wajibLengkap menentukan apakah field kosong langsung dianggap salah
// (POST dan PUT) atau memang boleh tidak dikirim (PATCH).
func validasiIsi(nim, name string, grade *float64, wajibLengkap bool) map[string]string {
	errs := map[string]string{}

	if nim == "" {
		if wajibLengkap {
			errs["nim"] = "wajib diisi"
		}
	} else if !nimValid(nim) {
		errs["nim"] = "harus berupa angka sepanjang 9 sampai 15 digit"
	}

	if name == "" {
		if wajibLengkap {
			errs["name"] = "wajib diisi"
		}
	} else if len(name) < 3 {
		errs["name"] = "minimal 3 karakter"
	} else if len(name) > 100 {
		errs["name"] = "maksimal 100 karakter"
	}

	if grade == nil {
		if wajibLengkap {
			errs["grade"] = "wajib diisi"
		}
	} else if *grade < 0 || *grade > 100 {
		errs["grade"] = "harus berada di rentang 0 sampai 100"
	}

	return errs
}
