package service

import (
	"errors"
	"fmt"
	"strings"

	"api-students/app/model"
)

var errUnknownEvent = errors.New("jenis event tidak dikenal")

// BuildMessage menyusun pesan dari sebuah event. Fungsi murni: tidak
// menyentuh database, Redis, maupun email, sehingga mudah diuji.
func BuildMessage(eventType string, e model.EnrollmentEvent, user model.Student) (model.Message, error) {
	course := e.CourseCode + " " + e.CourseName

	var subject, body string
	switch eventType {
	case model.EventEnrollmentCreated:
		subject = "KRS: berhasil terdaftar di " + course
		body = fmt.Sprintf("Halo %s, Anda tercatat sebagai peserta kelas %s.", user.Name, course)
	case model.EventEnrollmentDropped:
		subject = "KRS: keluar dari " + course
		body = fmt.Sprintf("Halo %s, Anda sudah tidak terdaftar di kelas %s.", user.Name, course)
	case model.EventEnrollmentSwapped:
		from := e.FromCourseCode + " " + e.FromCourseName
		subject = "KRS: pindah dari " + from + " ke " + course
		body = fmt.Sprintf("Halo %s, Anda telah pindah dari kelas %s ke kelas %s.", user.Name, from, course)
	default:
		return model.Message{}, fmt.Errorf("%w: %q", errUnknownEvent, eventType)
	}

	// Penyesuaian proyek: bila mahasiswa tidak mengisi email saat
	// mendaftar, alamat diturunkan dari NIM mengikuti konvensi email
	// mahasiswa universitas.
	to := user.Email
	if to == "" {
		to = strings.ToLower(user.NIM) + "@student.unair.ac.id"
	}

	return model.Message{To: to, Subject: subject, Body: body}, nil
}
