package model

import (
	"encoding/json"
	"time"
)

// Nama-nama event yang dicatat ke outbox. Ditulis sebagai konstanta,
// bukan string yang diketik ulang di banyak tempat, karena salah ketik pada
// nama event tidak ditangkap compiler maupun database.
const (
	EventEnrollmentCreated = "enrollment.created"
	EventEnrollmentDropped = "enrollment.dropped"
	EventEnrollmentSwapped = "enrollment.swapped"
)

// TaskSendNotification adalah nama JENIS task (task type); worker memakainya untuk
// memilih handler. QueueNotifications adalah nama QUEUE tempat task
// itu menunggu; worker hanya mengambil task dari queue yang ia dengarkan.
const (
	TaskSendNotification = "notification:send"
	QueueNotifications   = "notifications"
)

// EnrollmentEvent adalah isi sebuah event KRS.
//
// Nama dan kode kelas DISALIN ke dalam event, bukan hanya id-nya.
// Event adalah catatan tentang apa yang terjadi PADA SAAT ITU. Bila nama
// kelas diubah esok hari, notifikasi untuk event hari ini tetap memakai
// nama yang dilihat mahasiswa ketika ia mendaftar.
type EnrollmentEvent struct {
	UserID         int    `json:"user_id"`
	CourseID       int    `json:"course_id"`
	CourseCode     string `json:"course_code"`
	CourseName     string `json:"course_name"`
	FromCourseCode string `json:"from_course_code,omitempty"`
	FromCourseName string `json:"from_course_name,omitempty"`
}

// OutboxEvent adalah satu baris tabel outbox.
type OutboxEvent struct {
	ID        int64
	EventType string
	Payload   []byte
	CreatedAt time.Time
}

// NotificationJob adalah isi task yang dikirim ke queue.
//
// OutboxID ikut dibawa karena dialah idempotency key: worker memakainya
// untuk mengetahui apakah event ini sudah pernah dikirimkan.
//
// Payload dibiarkan sebagai JSON mentah (json.RawMessage). Relay cukup
// meneruskannya apa adanya tanpa perlu tahu bentuk setiap jenis event,
// yang membaca isinya adalah worker, berdasarkan EventType.
type NotificationJob struct {
	OutboxID  int64           `json:"outbox_id"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
}

// Message adalah pesan yang siap dikirim, tidak bergantung pada cara
// pengirimannya (email, push notification, dan seterusnya).
type Message struct {
	To      string
	Subject string
	Body    string
}

// Notification adalah notifikasi yang sudah terkirim, seperti yang dilihat
// mahasiswa pada GET /notifications/mine.
type Notification struct {
	ID        int64     `json:"id"`
	OutboxID  int64     `json:"outbox_id"`
	UserID    int       `json:"user_id"`
	Channel   string    `json:"channel"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	SentAt    time.Time `json:"sent_at"`
}

// QueueStats adalah ringkasan keadaan untuk admin.
type QueueStats struct {
	OutboxPending     int        `json:"outbox_pending"`
	OutboxPublished   int        `json:"outbox_published"`
	OldestPendingAge  string     `json:"oldest_pending_age"`
	NotificationsSent int        `json:"notifications_sent"`
	Queue             *QueueInfo `json:"queue"`
	QueueError        string     `json:"queue_error,omitempty"`
}

// QueueInfo adalah salinan sebagian informasi queue dari Asynq. Ditulis
// ulang di sini agar package model tetap tidak bergantung pada library apa
// pun.
type QueueInfo struct {
	Name      string `json:"name"`
	Pending   int    `json:"pending"`
	Active    int    `json:"active"`
	Scheduled int    `json:"scheduled"`
	Retry     int    `json:"retry"`
	Archived  int    `json:"archived"`
	Completed int    `json:"completed"`
	Processed int    `json:"processed_today"`
	Failed    int    `json:"failed_today"`
}
