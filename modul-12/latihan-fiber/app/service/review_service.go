package service

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

// ReviewService memakai DUA database sekaligus: data kelas dan KRS dibaca
// dari PostgreSQL, ulasan disimpan di MongoDB. Service-lah satu-satunya
// tempat yang mengenal keduanya.
type ReviewService struct {
	reviews repository.ReviewRepository
	courses repository.CourseRepository
	perms   *helper.PermissionSet
}

func NewReviewService(
	reviews repository.ReviewRepository,
	courses repository.CourseRepository,
	perms *helper.PermissionSet,
) *ReviewService {
	return &ReviewService{reviews: reviews, courses: courses, perms: perms}
}

// ---------- POST /courses/:id/reviews ----------

// Create godoc
//
// @Summary      Menulis ulasan untuk sebuah kelas
// @Description  Hanya peserta kelas. Satu mahasiswa hanya boleh menulis satu ulasan per kelas.
// @Tags         reviews
// @Accept       json
// @Produce      json
// @Param        id    path  int                        true  "ID kelas"
// @Param        body  body  model.CreateReviewRequest  true  "Isi ulasan"
// @Success      201   {object}  model.WebResponse{data=model.Review}
// @Failure      403   {object}  model.ErrorResponse  "NOT_ENROLLED"
// @Failure      409   {object}  model.ErrorResponse  "ALREADY_REVIEWED"
// @Failure      422   {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /courses/{id}/reviews [post]
func (s *ReviewService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	courseID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id kelas harus berupa angka positif")
	}

	var req model.CreateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	req.Comment = strings.TrimSpace(req.Comment)
	req.Tags = NormalizeTags(req.Tags)
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	// Tidak ada FOREIGN KEY yang menghubungkan MongoDB dengan PostgreSQL.
	// Pemeriksaan "kelasnya ada" dan "mahasiswa ini peserta kelasnya" harus
	// dilakukan sendiri oleh service, dengan bertanya ke PostgreSQL.
	if _, err := s.courses.FindByID(ctx, courseID); err != nil {
		return translateReviewError(err)
	}
	enrolled, err := s.courses.IsEnrolled(ctx, courseID, current.UserID)
	if err != nil {
		return helper.Internal(err)
	}
	if !enrolled {
		return helper.NewError(fiber.StatusForbidden, "NOT_ENROLLED",
			"hanya peserta kelas yang boleh menulis ulasan")
	}

	created, err := s.reviews.Create(ctx, model.Review{
		CourseID: courseID, UserID: current.UserID, Username: current.Username,
		Rating: req.Rating, Comment: req.Comment, Tags: req.Tags,
	})
	if err != nil {
		return translateReviewError(err)
	}
	return helper.Created(c, "ulasan berhasil disimpan", created, "/api/v1/reviews/"+created.ID)
}

// ---------- GET /courses/:id/reviews ----------

// ListByCourse godoc
//
// @Summary      Daftar ulasan sebuah kelas, terbaru lebih dulu
// @Description  Cursor pagination: kirim meta.next_before sebagai parameter before untuk halaman berikutnya.
// @Tags         reviews
// @Produce      json
// @Param        id      path   int     true   "ID kelas"
// @Param        tag     query  string  false  "Saring berdasarkan tag"
// @Param        before  query  string  false  "Cursor: id ulasan terakhir yang sudah dilihat"
// @Param        limit   query  int     false  "Jumlah per halaman, 1 sampai 50"  default(10)
// @Success      200     {object}  model.ReviewPage
// @Failure      400     {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /courses/{id}/reviews [get]
func (s *ReviewService) ListByCourse(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	courseID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id kelas harus berupa angka positif")
	}

	q := model.ReviewQuery{
		CourseID: courseID,
		Tag:      strings.ToLower(strings.TrimSpace(c.Query("tag"))),
		Before:   c.Query("before"),
		Limit:    c.QueryInt("limit", 10),
	}
	if q.Limit < 1 || q.Limit > 50 {
		q.Limit = 10
	}

	// Minta satu lebih banyak untuk mengetahui apakah masih ada halaman
	// berikutnya — teknik yang sama dengan Modul 7.
	want := q.Limit
	q.Limit = want + 1
	reviews, err := s.reviews.FindByCourse(ctx, q)
	if err != nil {
		return translateReviewError(err)
	}

	hasMore := len(reviews) > want
	if hasMore {
		reviews = reviews[:want]
	}
	next := ""
	if hasMore {
		next = reviews[len(reviews)-1].ID
	}

	return c.JSON(model.ReviewPage{
		Success: true,
		Message: "daftar ulasan berhasil diambil",
		Data:    reviews,
		Meta:    model.ReviewPageMeta{Limit: want, HasMore: hasMore, NextBefore: next},
	})
}

// ---------- GET /courses/:id/reviews/summary ----------

// Summary godoc
//
// @Summary      Ringkasan ulasan sebuah kelas
// @Description  Statistik umum, distribusi rating, dan tag terpopuler — hasil aggregation pipeline MongoDB.
// @Tags         reviews
// @Produce      json
// @Param        id  path  int  true  "ID kelas"
// @Success      200  {object}  model.WebResponse{data=model.ReviewSummary}
// @Failure      400  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse  "Kelas tidak ada"
// @Security     BearerAuth
// @Router       /courses/{id}/reviews/summary [get]
func (s *ReviewService) Summary(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	courseID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id kelas harus berupa angka positif")
	}
	if _, err := s.courses.FindByID(ctx, courseID); err != nil {
		return translateReviewError(err)
	}

	summary, err := s.reviews.Summary(ctx, courseID)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "ringkasan ulasan berhasil diambil", summary)
}

// ---------- GET /reviews/:id ----------

// Get godoc
//
// @Summary  Detail satu ulasan
// @Tags     reviews
// @Produce  json
// @Param    id  path  string  true  "ID ulasan (ObjectID)"
// @Success  200  {object}  model.WebResponse{data=model.Review}
// @Failure  400  {object}  model.ErrorResponse  "ID ulasan tidak valid"
// @Failure  404  {object}  model.ErrorResponse
// @Security BearerAuth
// @Router   /reviews/{id} [get]
func (s *ReviewService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	review, err := s.reviews.FindByID(ctx, c.Params("id"))
	if err != nil {
		return translateReviewError(err)
	}
	return helper.Success(c, fiber.StatusOK, "ulasan ditemukan", review)
}

// ---------- PATCH /reviews/:id ----------

// Update godoc
//
// @Summary      Mengubah ulasan milik sendiri
// @Description  Hanya field yang dikirim yang berubah. Pemilik ulasan ditentukan di dalam filter MongoDB, bukan di Go.
// @Tags         reviews
// @Accept       json
// @Produce      json
// @Param        id    path  string                     true  "ID ulasan (ObjectID)"
// @Param        body  body  model.UpdateReviewRequest  true  "Field yang ingin diubah"
// @Success      200   {object}  model.WebResponse{data=model.Review}
// @Failure      400   {object}  model.ErrorResponse  "ID ulasan tidak valid, body bukan JSON, atau tidak ada field yang diubah"
// @Failure      403   {object}  model.ErrorResponse  "Bukan pemilik ulasan"
// @Failure      404   {object}  model.ErrorResponse
// @Failure      422   {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /reviews/{id} [patch]
func (s *ReviewService) Update(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	var req model.UpdateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if req.Rating == nil && req.Comment == nil && req.Tags == nil {
		return helper.BadRequest("tidak ada field yang diubah")
	}
	if req.Comment != nil {
		trimmed := strings.TrimSpace(*req.Comment)
		req.Comment = &trimmed
	}
	if req.Tags != nil {
		normalized := NormalizeTags(*req.Tags)
		req.Tags = &normalized
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	updated, err := s.reviews.UpdateOwn(ctx, c.Params("id"), current.UserID, req)
	if err != nil {
		return translateReviewError(err)
	}
	return helper.Success(c, fiber.StatusOK, "ulasan berhasil diubah", updated)
}

// ---------- DELETE /reviews/:id ----------
// Pemilik boleh menghapus ulasannya sendiri; pemegang permission
// review:moderate boleh menghapus ulasan siapa pun.

// Delete godoc
//
// @Summary      Menghapus ulasan
// @Description  Pemilik ulasan selalu boleh; pemegang permission review:moderate boleh menghapus ulasan siapa pun.
// @Tags         reviews
// @Produce      json
// @Param        id  path  string  true  "ID ulasan (ObjectID)"
// @Success      204  "Berhasil dihapus; tanpa body"
// @Failure      400  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse  "Bukan pemilik ulasan"
// @Failure      404  {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /reviews/{id} [delete]
func (s *ReviewService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	anyOwner := s.perms.Can(current.Role, "review:moderate")
	if err := s.reviews.Delete(ctx, c.Params("id"), current.UserID, anyOwner); err != nil {
		return translateReviewError(err)
	}
	return helper.NoContent(c)
}

// ---------- POST /reviews/:id/replies ----------

// Reply godoc
//
// @Summary      Membalas sebuah ulasan
// @Description  Balasan disimpan di dalam dokumen ulasan (embedding); jumlahnya dibatasi 50 secara atomik.
// @Tags         reviews
// @Accept       json
// @Produce      json
// @Param        id    path  string              true  "ID ulasan (ObjectID)"
// @Param        body  body  model.ReplyRequest  true  "Isi balasan"
// @Success      201   {object}  model.WebResponse{data=model.Review}
// @Failure      400   {object}  model.ErrorResponse
// @Failure      404   {object}  model.ErrorResponse
// @Failure      409   {object}  model.ErrorResponse  "REPLY_LIMIT: jumlah balasan sudah mencapai batas"
// @Failure      422   {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /reviews/{id}/replies [post]
func (s *ReviewService) Reply(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	var req model.ReplyRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	req.Body = strings.TrimSpace(req.Body)
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	updated, err := s.reviews.AddReply(ctx, c.Params("id"), model.Reply{
		UserID: current.UserID, Username: current.Username, Body: req.Body,
	})
	if err != nil {
		return translateReviewError(err)
	}
	return helper.Success(c, fiber.StatusCreated, "balasan berhasil ditambahkan", updated)
}

// ---------- POST /reviews/:id/helpful ----------

// MarkHelpful godoc
//
// @Summary      Menilai ulasan membantu
// @Description  Satu mahasiswa hanya boleh menilai satu kali per ulasan; penilaian ulasan sendiri ditolak.
// @Tags         reviews
// @Produce      json
// @Param        id  path  string  true  "ID ulasan (ObjectID)"
// @Success      200  {object}  model.WebResponse{data=model.Review}
// @Failure      400  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Failure      409  {object}  model.ErrorResponse  "OWN_REVIEW atau ALREADY_VOTED"
// @Security     BearerAuth
// @Router       /reviews/{id}/helpful [post]
func (s *ReviewService) MarkHelpful(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	updated, err := s.reviews.MarkHelpful(ctx, c.Params("id"), current.UserID)
	if err != nil {
		return translateReviewError(err)
	}
	return helper.Success(c, fiber.StatusOK, "terima kasih atas penilaiannya", updated)
}

func translateReviewError(err error) error {
	switch {
	case errors.Is(err, repository.ErrInvalidID):
		return helper.BadRequest("id ulasan tidak valid")
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("data tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.NewError(fiber.StatusConflict, "ALREADY_REVIEWED",
			"Anda sudah menulis ulasan untuk kelas ini, ubah ulasan yang ada")
	case errors.Is(err, repository.ErrNotOwner):
		return helper.Forbidden("hanya pemilik ulasan yang boleh melakukan ini")
	case errors.Is(err, repository.ErrOwnReview):
		return helper.NewError(fiber.StatusConflict, "OWN_REVIEW", "tidak dapat menilai ulasan sendiri")
	case errors.Is(err, repository.ErrAlreadyVoted):
		return helper.NewError(fiber.StatusConflict, "ALREADY_VOTED", "Anda sudah menilai ulasan ini")
	case errors.Is(err, repository.ErrReplyLimit):
		return helper.NewError(fiber.StatusConflict, "REPLY_LIMIT", "jumlah balasan sudah mencapai batas")
	default:
		return helper.Internal(err)
	}
}