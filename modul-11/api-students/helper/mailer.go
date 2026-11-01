package helper

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"api-students/app/model"
)

// Dua golongan kegagalan pengiriman. Pembedaan ini menentukan apakah
// sebuah task layak di-retry.
var (
	// ErrTemporaryDelivery: server email sedang sibuk atau jaringan putus.
	// Melakukan retry beberapa saat kemudian kemungkinan besar berhasil.
	ErrTemporaryDelivery = errors.New("pengiriman gagal sementara")

	// ErrPermanentDelivery: alamat tujuan tidak ada. Melakukan retry seribu
	// kali pun hasilnya akan tetap sama.
	ErrPermanentDelivery = errors.New("pengiriman gagal permanen")
)

// Mailer adalah kemampuan mengirim pesan. Service hanya mengenal interface
// ini, sehingga LogMailer dapat diganti mailer SMTP sungguhan tanpa
// mengubah satu baris pun kode service.
type Mailer interface {
	Send(ctx context.Context, msg model.Message) error
}

// LogMailer adalah mailer tiruan untuk praktikum. Ia tidak mengirim email
// sungguhan, tetapi menulis pesannya ke logs/mail.log dan meniru dua sifat
// server email sungguhan: lambat, dan sesekali gagal.
type LogMailer struct {
	mu        sync.Mutex
	path      string
	delay     time.Duration
	failEvery int64
	attempts  atomic.Int64
}

// NewLogMailer membuat LogMailer.
//
//   - delay     : lama satu kali pengiriman, meniru server email yang lambat.
//   - failEvery : setiap attempt ke-N gagal sementara. Nol berarti tidak
//     pernah gagal.
func NewLogMailer(path string, delay time.Duration, failEvery int) (Mailer, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("menyiapkan folder log email: %w", err)
	}
	// Pointer, bukan nilai: mutex dan counter attempt harus dimiliki
	// bersama oleh seluruh goroutine worker, bukan disalin per pemanggilan.
	return &LogMailer{path: path, delay: delay, failEvery: int64(failEvery)}, nil
}

func (m *LogMailer) Send(ctx context.Context, msg model.Message) error {
	n := m.attempts.Add(1)

	// Meniru waktu tempuh ke server email. Bila context dibatalkan lebih
	// dulu (timeout task habis, atau worker sedang dimatikan),
	// pengiriman dihentikan saat itu juga.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(m.delay):
	}

	// Domain .invalid dicadangkan oleh RFC 2606 dan dijamin tidak pernah
	// ada. Kita memakainya untuk meniru "alamat tujuan tidak ditemukan".
	if strings.HasSuffix(strings.ToLower(msg.To), ".invalid") {
		return fmt.Errorf("%w: alamat %s tidak ditemukan", ErrPermanentDelivery, msg.To)
	}
	if m.failEvery > 0 && n%m.failEvery == 0 {
		return fmt.Errorf("%w: server email sibuk (attempt ke-%d)", ErrTemporaryDelivery, n)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	f, err := os.OpenFile(m.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrTemporaryDelivery, err)
	}
	defer f.Close()

	// Hanya alamat dan subjek yang dicatat. Isi pesan dapat memuat data
	// pribadi yang tidak perlu ikut tersimpan di file log.
	_, err = fmt.Fprintf(f, "%s  to=%s  subject=%q\n",
		time.Now().Format(time.RFC3339), msg.To, msg.Subject)
	return err
}
