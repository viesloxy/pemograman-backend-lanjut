package repository

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"api-students/app/model"
)

var (
	ErrInvalidID    = errors.New("format id tidak valid")
	ErrNotOwner     = errors.New("bukan pemilik data")
	ErrOwnReview    = errors.New("tidak boleh menilai ulasan sendiri")
	ErrAlreadyVoted = errors.New("sudah pernah menilai ulasan ini")
	ErrReplyLimit   = errors.New("jumlah balasan sudah mencapai batas")
)

// MaxRepliesPerReview membatasi pertumbuhan array replies. Satu dokumen
// MongoDB maksimal 16 MB; array yang dibiarkan tumbuh tanpa batas suatu saat
// akan menabrak batas itu.
const MaxRepliesPerReview = 50

// reviewDocument adalah bentuk ulasan SEBAGAIMANA TERSIMPAN.
//
// Tag bson menentukan nama field di database; tag json tidak diperlukan karena
// struct ini tidak pernah dikirim ke client. Pemisahan ini membuat package
// model tetap bebas dari tipe-tipe milik driver.
type reviewDocument struct {
	ID           bson.ObjectID   `bson:"_id,omitempty"`
	CourseID     int             `bson:"course_id"`
	UserID       int             `bson:"user_id"`
	Username     string          `bson:"username"`
	Rating       int             `bson:"rating"`
	Comment      string          `bson:"comment"`
	Tags         []string        `bson:"tags"`
	Replies      []replyDocument `bson:"replies"`
	HelpfulCount int             `bson:"helpful_count"`
	HelpfulBy    []int           `bson:"helpful_by"`
	CreatedAt    time.Time       `bson:"created_at"`
	UpdatedAt    time.Time       `bson:"updated_at"`
}

type replyDocument struct {
	ID        bson.ObjectID `bson:"_id"`
	UserID    int           `bson:"user_id"`
	Username  string        `bson:"username"`
	Body      string        `bson:"body"`
	CreatedAt time.Time     `bson:"created_at"`
}

func (d reviewDocument) toModel() model.Review {
	replies := make([]model.Reply, 0, len(d.Replies))
	for _, r := range d.Replies {
		replies = append(replies, model.Reply{
			ID: r.ID.Hex(), UserID: r.UserID, Username: r.Username,
			Body: r.Body, CreatedAt: r.CreatedAt,
		})
	}
	tags := d.Tags
	if tags == nil {
		tags = []string{}
	}
	return model.Review{
		ID: d.ID.Hex(), CourseID: d.CourseID, UserID: d.UserID, Username: d.Username,
		Rating: d.Rating, Comment: d.Comment, Tags: tags, Replies: replies,
		HelpfulCount: d.HelpfulCount, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

type reviewMongoRepository struct{ coll *mongo.Collection }

// now membulatkan waktu ke milidetik. Tipe date pada BSON hanya menyimpan
// sampai milidetik, sedangkan time.Time di Go menyimpan sampai nanodetik.
// Tanpa pembulatan, nilai yang dikirim ke client saat membuat ulasan akan
// berbeda dari nilai yang dibaca kembali dari database.
func now() time.Time { return time.Now().UTC().Truncate(time.Millisecond) }

func parseID(id string) (bson.ObjectID, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return bson.ObjectID{}, ErrInvalidID
	}
	return oid, nil
}
