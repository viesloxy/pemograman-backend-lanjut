package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

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

// ReviewRepository adalah kontrak ulasan bagi layer service. Layer di atasnya
// tidak perlu tahu bahwa data ini tersimpan di MongoDB, bukan di PostgreSQL.
type ReviewRepository interface {
	EnsureIndexes(ctx context.Context) error
	Create(ctx context.Context, r model.Review) (model.Review, error)
	FindByID(ctx context.Context, id string) (model.Review, error)
	FindByCourse(ctx context.Context, q model.ReviewQuery) ([]model.Review, error)
	UpdateOwn(ctx context.Context, id string, userID int, req model.UpdateReviewRequest) (model.Review, error)
	Delete(ctx context.Context, id string, userID int, anyOwner bool) error
	AddReply(ctx context.Context, id string, reply model.Reply) (model.Review, error)
	MarkHelpful(ctx context.Context, id string, userID int) (model.Review, error)
	Summary(ctx context.Context, courseID int) (model.ReviewSummary, error)
}

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

func NewReviewRepository(db *mongo.Database) ReviewRepository {
	return &reviewMongoRepository{coll: db.Collection("course_reviews")}
}

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

// EnsureIndexes dijalankan sekali saat aplikasi menyala. Membuat index yang
// sudah ada tidak melakukan apa pun, sehingga aman dipanggil berulang kali.
func (r *reviewMongoRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			// Satu mahasiswa hanya boleh menulis SATU ulasan per kelas.
			// Aturan ini dijaga database, bukan hanya diperiksa di kode.
			Keys:    bson.D{{Key: "course_id", Value: 1}, {Key: "user_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("course_user_unique"),
		},
		{
			// Mendukung daftar ulasan satu kelas, terbaru lebih dulu.
			Keys:    bson.D{{Key: "course_id", Value: 1}, {Key: "_id", Value: -1}},
			Options: options.Index().SetName("course_newest"),
		},
		{
			// Mendukung filter berdasarkan tag. Index pada field array
			// disebut multikey index: setiap elemen array diindeks.
			Keys:    bson.D{{Key: "course_id", Value: 1}, {Key: "tags", Value: 1}},
			Options: options.Index().SetName("course_tags"),
		},
	})
	if err != nil {
		return fmt.Errorf("membuat index course_reviews: %w", err)
	}
	return nil
}

func (r *reviewMongoRepository) Create(ctx context.Context, rv model.Review) (model.Review, error) {
	t := now()
	doc := reviewDocument{
		ID:        bson.NewObjectID(),
		CourseID: rv.CourseID, UserID: rv.UserID, Username: rv.Username,
		Rating: rv.Rating, Comment: rv.Comment, Tags: rv.Tags,
		Replies: []replyDocument{}, HelpfulBy: []int{},
		CreatedAt: t, UpdatedAt: t,
	}
	if doc.Tags == nil {
		doc.Tags = []string{}
	}

	if _, err := r.coll.InsertOne(ctx, doc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return model.Review{}, ErrDuplicate
		}
		return model.Review{}, fmt.Errorf("menyimpan ulasan: %w", err)
	}
	return doc.toModel(), nil
}

func (r *reviewMongoRepository) FindByID(ctx context.Context, id string) (model.Review, error) {
	oid, err := parseID(id)
	if err != nil {
		return model.Review{}, err
	}

	var doc reviewDocument
	if err := r.coll.FindOne(ctx, bson.M{"_id": oid}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Review{}, ErrNotFound
		}
		return model.Review{}, fmt.Errorf("mengambil ulasan: %w", err)
	}
	return doc.toModel(), nil
}

// FindByCourse mengambil ulasan satu kelas, terbaru lebih dulu.
//
// ObjectID memuat waktu pembuatannya, sehingga mengurutkan berdasarkan _id
// sama dengan mengurutkan berdasarkan waktu dibuat — dan _id dijamin unik.
// Itulah sebabnya _id dapat langsung dipakai sebagai cursor seperti Modul 7.
func (r *reviewMongoRepository) FindByCourse(ctx context.Context, q model.ReviewQuery) ([]model.Review, error) {
	filter := bson.M{"course_id": q.CourseID}

	// Nilai q.Tag selalu bertipe string. Ia tidak mungkin berubah menjadi
	// operator seperti {"$ne": null}, karena tipenya sudah dipastikan oleh Go.
	if q.Tag != "" {
		filter["tags"] = q.Tag
	}
	if q.Before != "" {
		before, err := parseID(q.Before)
		if err != nil {
			return nil, err
		}
		filter["_id"] = bson.M{"$lt": before}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "_id", Value: -1}}).
		SetLimit(int64(q.Limit)).
		// helpful_by tidak pernah dikirim ke client; tidak perlu diambil.
		SetProjection(bson.M{"helpful_by": 0})

	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar ulasan: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []reviewDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("membaca daftar ulasan: %w", err)
	}

	result := make([]model.Review, 0, len(docs))
	for _, d := range docs {
		result = append(result, d.toModel())
	}
	return result, nil
}

// Method berikut diisi pada Langkah 6 dan Langkah 7 modul.

func (r *reviewMongoRepository) UpdateOwn(ctx context.Context, id string, userID int, req model.UpdateReviewRequest) (model.Review, error) {
	return model.Review{}, errBelumDiimplementasi
}

func (r *reviewMongoRepository) Delete(ctx context.Context, id string, userID int, anyOwner bool) error {
	return errBelumDiimplementasi
}

func (r *reviewMongoRepository) AddReply(ctx context.Context, id string, reply model.Reply) (model.Review, error) {
	return model.Review{}, errBelumDiimplementasi
}

func (r *reviewMongoRepository) MarkHelpful(ctx context.Context, id string, userID int) (model.Review, error) {
	return model.Review{}, errBelumDiimplementasi
}

func (r *reviewMongoRepository) Summary(ctx context.Context, courseID int) (model.ReviewSummary, error) {
	return model.ReviewSummary{}, errBelumDiimplementasi
}

var errBelumDiimplementasi = errors.New("belum diimplementasi")
