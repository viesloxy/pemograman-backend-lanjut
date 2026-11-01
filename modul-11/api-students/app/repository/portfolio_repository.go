package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"api-students/app/model"
)

var ErrProjectNotFound = errors.New("proyek tidak ditemukan")
var ErrProjectLimit = errors.New("jumlah proyek sudah mencapai batas")

// MaxProjectsPerPortfolio membatasi pertumbuhan array projects. Alasan yang
// sama dengan replies pada ulasan: dokumen MongoDB maksimal 16 MB.
const MaxProjectsPerPortfolio = 20

// PortfolioRepository adalah kontrak portofolio bagi layer service.
type PortfolioRepository interface {
	EnsureIndexes(ctx context.Context) error
	GetByStudent(ctx context.Context, studentID int) (model.Portfolio, error)
	Replace(ctx context.Context, p model.Portfolio) (model.Portfolio, error)
	AddProject(ctx context.Context, studentID int, pr model.Project) (model.Portfolio, error)
	RemoveProject(ctx context.Context, studentID int, projectID string) (model.Portfolio, error)
	StatsTechnologies(ctx context.Context) ([]model.TechnologyCount, error)
}

// portfolioDocument adalah bentuk portofolio SEBAGAIMANA TERSIMPAN.
type portfolioDocument struct {
	ID           bson.ObjectID        `bson:"_id,omitempty"`
	StudentID    int                  `bson:"student_id"`
	Headline     string               `bson:"headline"`
	Bio          string               `bson:"bio"`
	Projects     []projectDocument    `bson:"projects"`
	Certificates []certificateDocument `bson:"certificates"`
	CreatedAt    time.Time            `bson:"created_at"`
	UpdatedAt    time.Time            `bson:"updated_at"`
}

type projectDocument struct {
	ID           bson.ObjectID `bson:"_id"`
	Title        string        `bson:"title"`
	Technologies []string      `bson:"technologies"`
	Link         string        `bson:"link"`
	Year         int           `bson:"year"`
}

type certificateDocument struct {
	Name     string `bson:"name"`
	Issuer   string `bson:"issuer"`
	IssuedAt string `bson:"issued_at"`
}

func (d portfolioDocument) toModel() model.Portfolio {
	projects := make([]model.Project, 0, len(d.Projects))
	for _, p := range d.Projects {
		tech := p.Technologies
		if tech == nil {
			tech = []string{}
		}
		projects = append(projects, model.Project{
			ID: p.ID.Hex(), Title: p.Title, Technologies: tech,
			Link: p.Link, Year: p.Year,
		})
	}
	certs := make([]model.Certificate, 0, len(d.Certificates))
	for _, c := range d.Certificates {
		certs = append(certs, model.Certificate{Name: c.Name, Issuer: c.Issuer, IssuedAt: c.IssuedAt})
	}
	return model.Portfolio{
		StudentID: d.StudentID, Headline: d.Headline, Bio: d.Bio,
		Projects: projects, Certificates: certs,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

type portfolioMongoRepository struct{ coll *mongo.Collection }

func NewPortfolioRepository(db *mongo.Database) PortfolioRepository {
	return &portfolioMongoRepository{coll: db.Collection("student_portfolios")}
}

// EnsureIndexes membuat index penjaga aturan. Unique index student_id-lah
// yang menjamin "satu mahasiswa satu portofolio" — dua permintaan serentak
// yang mencoba membuat portofolio untuk mahasiswa yang sama tidak mungkin
// sama-sama berhasil, apa pun urutan kejadiannya.
func (r *portfolioMongoRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "student_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("student_unique"),
		},
	})
	if err != nil {
		return fmt.Errorf("membuat index student_portfolios: %w", err)
	}
	return nil
}

func newPortfolioDocument(p model.Portfolio) portfolioDocument {
	projects := make([]projectDocument, 0, len(p.Projects))
	for _, pr := range p.Projects {
		projects = append(projects, projectDocument{
			ID: bson.NewObjectID(), Title: pr.Title, Technologies: pr.Technologies,
			Link: pr.Link, Year: pr.Year,
		})
	}
	certs := make([]certificateDocument, 0, len(p.Certificates))
	for _, c := range p.Certificates {
		certs = append(certs, certificateDocument{Name: c.Name, Issuer: c.Issuer, IssuedAt: c.IssuedAt})
	}
	return portfolioDocument{
		StudentID: p.StudentID, Headline: p.Headline, Bio: p.Bio,
		Projects: projects, Certificates: certs,
	}
}

// GetByStudent mengambil portofolio satu mahasiswa.
func (r *portfolioMongoRepository) GetByStudent(ctx context.Context, studentID int) (model.Portfolio, error) {
	var doc portfolioDocument
	if err := r.coll.FindOne(ctx, bson.M{"student_id": studentID}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Portfolio{}, ErrNotFound
		}
		return model.Portfolio{}, fmt.Errorf("mengambil portofolio: %w", err)
	}
	return doc.toModel(), nil
}

// Replace membuat ATAU mengganti seluruh portofolio mahasiswa dalam satu
// operasi upsert atomik. $setOnInsert hanya bekerja saat dokumen baru dibuat,
// sehingga created_at dan _id tidak pernah tertimpa oleh pembaruan berikutnya.
func (r *portfolioMongoRepository) Replace(ctx context.Context, p model.Portfolio) (model.Portfolio, error) {
	doc := newPortfolioDocument(p)

	_, err := r.coll.UpdateOne(ctx,
		bson.M{"student_id": p.StudentID},
		bson.M{
			"$set": bson.M{
				"headline":     doc.Headline,
				"bio":          doc.Bio,
				"projects":     doc.Projects,
				"certificates": doc.Certificates,
				"updated_at":   now(),
			},
			"$setOnInsert": bson.M{
				"_id":        bson.NewObjectID(),
				"student_id": p.StudentID,
				"created_at": now(),
			},
		},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		return model.Portfolio{}, fmt.Errorf("menyimpan portofolio: %w", err)
	}
	return r.GetByStudent(ctx, p.StudentID)
}

// AddProject menambah satu proyek ke array projects.
//
// Syarat "projects.19 tidak ada" berarti array belum berisi 20 elemen.
// Pemeriksaan batas dan penambahan terjadi dalam satu operasi, sehingga
// sepuluh permintaan serentak ketika tinggal dua tempat tidak mungkin
// sama-sama lolos — pelajaran Langkah 6 dipakai ulang di sini.
func (r *portfolioMongoRepository) AddProject(ctx context.Context, studentID int, pr model.Project) (model.Portfolio, error) {
	doc := projectDocument{
		ID: bson.NewObjectID(), Title: pr.Title, Technologies: pr.Technologies,
		Link: pr.Link, Year: pr.Year,
	}
	limitField := "projects." + strconv.Itoa(MaxProjectsPerPortfolio-1)

	var updated portfolioDocument
	err := r.coll.FindOneAndUpdate(ctx,
		bson.M{"student_id": studentID, limitField: bson.M{"$exists": false}},
		bson.M{"$push": bson.M{"projects": doc}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updated)
	if err == nil {
		return updated.toModel(), nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return model.Portfolio{}, fmt.Errorf("menambah proyek: %w", err)
	}

	count, err := r.coll.CountDocuments(ctx, bson.M{"student_id": studentID})
	if err != nil {
		return model.Portfolio{}, fmt.Errorf("memeriksa portofolio: %w", err)
	}
	if count == 0 {
		return model.Portfolio{}, ErrNotFound
	}
	return model.Portfolio{}, ErrProjectLimit
}

// RemoveProject menghapus satu proyek dengan $pull dan mengembalikan
// portofolio terbaru. Filter memuat projects._id sehingga "portofolio tidak
// ada" dan "proyeknya tidak ada" dapat dibedakan.
func (r *portfolioMongoRepository) RemoveProject(ctx context.Context, studentID int, projectID string) (model.Portfolio, error) {
	pid, err := parseID(projectID)
	if err != nil {
		return model.Portfolio{}, err
	}

	var updated portfolioDocument
	err = r.coll.FindOneAndUpdate(ctx,
		bson.M{"student_id": studentID, "projects._id": pid},
		bson.M{
			"$pull":      bson.M{"projects": bson.M{"_id": pid}},
			"$set":       bson.M{"updated_at": now()},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updated)
	if err == nil {
		return updated.toModel(), nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return model.Portfolio{}, fmt.Errorf("menghapus proyek: %w", err)
	}

	count, err := r.coll.CountDocuments(ctx, bson.M{"student_id": studentID})
	if err != nil {
		return model.Portfolio{}, fmt.Errorf("memeriksa portofolio: %w", err)
	}
	if count == 0 {
		return model.Portfolio{}, ErrNotFound
	}
	return model.Portfolio{}, ErrProjectNotFound
}

// StatsTechnologies meringkas sepuluh teknologi paling banyak dipakai di
// seluruh portofolio. Dua kali $unwind diperlukan: pertama memecah array
// projects, kedua memecah array technologies di dalamnya.
func (r *portfolioMongoRepository) StatsTechnologies(ctx context.Context) ([]model.TechnologyCount, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$unwind", Value: "$projects"}},
		{{Key: "$unwind", Value: "$projects.technologies"}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$projects.technologies",
			"total": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.D{
			{Key: "total", Value: -1},
			{Key: "_id", Value: 1},
		}}},
		{{Key: "$limit", Value: 10}},
	}

	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("meringkas teknologi: %w", err)
	}
	defer cursor.Close(ctx)

	var out []struct {
		Technology string `bson:"_id"`
		Total      int    `bson:"total"`
	}
	if err := cursor.All(ctx, &out); err != nil {
		return nil, fmt.Errorf("membaca ringkasan teknologi: %w", err)
	}

	result := make([]model.TechnologyCount, 0, len(out))
	for _, t := range out {
		result = append(result, model.TechnologyCount{Technology: t.Technology, Total: t.Total})
	}
	return result, nil
}
