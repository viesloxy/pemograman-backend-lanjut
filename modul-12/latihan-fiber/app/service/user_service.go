package service

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

type UserService struct {
	repo  repository.UserRepository
	perms *helper.PermissionSet
}

func NewUserService(
	repo repository.UserRepository,
	perms *helper.PermissionSet,
) *UserService {
	return &UserService{repo: repo, perms: perms}
}

// List godoc
//
// @Summary      Daftar seluruh user
// @Description  Cursor pagination: kirim meta.next_cursor sebagai parameter cursor untuk halaman berikutnya.
// @Description  Format response dapat dipilih lewat header Accept: application/json atau text/csv.
// @Description  Membutuhkan permission user:list.
// @Tags         users
// @Produce      json
// @Param        limit      query  int     false  "Jumlah per halaman, 1 sampai 50"  default(10)
// @Param        search     query  string  false  "Saring username atau email yang mengandung teks ini"
// @Param        is_active  query  bool    false  "Saring menurut status aktif"
// @Param        cursor     query  string  false  "Cursor: meta.next_cursor dari halaman sebelumnya"
// @Success      200        {object}  model.WebResponse{data=[]model.User}
// @Failure      400        {object}  model.ErrorResponse  "Cursor atau is_active tidak valid"
// @Failure      401        {object}  model.ErrorResponse
// @Failure      403        {object}  model.ErrorResponse
// @Failure      406        {object}  model.ErrorResponse  "Format Accept tidak tersedia"
// @Security     BearerAuth
// @Router       /users [get]
func (s *UserService) List(c *fiber.Ctx) error {
	// Format dipilih SEBELUM query dijalankan.
	format, err := helper.Negotiate(c, helper.FormatJSON, helper.FormatCSV)
	if err != nil {
		return err
	}

	q, err := helper.ParseCursorQuery(c)
	if err != nil {
		return err
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	rows, err := s.repo.FindAfterCursor(ctx, q)
	if err != nil {
		return helper.Internal(err)
	}

	// Baris tambahan hasil limit+1 dipotong di sini.
	hasMore := len(rows) > q.Limit
	if hasMore {
		rows = rows[:q.Limit]
	}

	if format == helper.FormatCSV {
		return helper.WriteUsersCSV(c, rows)
	}

	meta := &model.CursorMeta{Limit: q.Limit, HasMore: hasMore}
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		meta.NextCursor = helper.EncodeCursor(last.CreatedAt, last.ID)
	}

	return helper.SuccessCursor(c, "daftar user berhasil diambil", rows, meta)
}

// ---------- GET /users/:id ----------

// Get godoc
//
// @Summary  Detail satu user
// @Description  Pemilik data boleh melihat data sendiri; user lain membutuhkan permission user:read:any.
// @Tags     users
// @Produce  json
// @Param    id  path  int  true  "ID user"
// @Success  200  {object}  model.WebResponse{data=model.User}
// @Failure  400  {object}  model.ErrorResponse  "ID bukan angka positif"
// @Failure  403  {object}  model.ErrorResponse
// @Failure  404  {object}  model.ErrorResponse
// @Security BearerAuth
// @Router   /users/{id} [get]
func (s *UserService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	// Pemeriksaan hak akses dilakukan SEBELUM data diambil.
	if !CanAccessUser(current, id, s.perms, "user:read:any") {
		return helper.Forbidden("tidak berhak mengakses data user lain")
	}

	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(err, "user")
	}
	return helper.Success(c, fiber.StatusOK, "user ditemukan", user)
}

// Create godoc
//
// @Summary      Membuat akun baru (admin)
// @Description  Membutuhkan permission user:update:any. Berbeda dengan /auth/register, role dapat dipilih dan password di-hash di sini.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body  model.CreateUserRequest  true  "Data akun baru"
// @Success      201   {object}  model.WebResponse{data=model.User}
// @Header       201   {string}  Location  "URL user yang baru dibuat"
// @Failure      400   {object}  model.ErrorResponse  "Body bukan JSON yang valid"
// @Failure      401   {object}  model.ErrorResponse
// @Failure      403   {object}  model.ErrorResponse
// @Failure      409   {object}  model.ErrorResponse  "Username sudah dipakai"
// @Failure      422   {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /users [post]
func (s *UserService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)

	newUser, err := s.repo.Create(ctx, model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		IsActive: true,
	})
	if err != nil {
		return translateError(err, "user")
	}

	return helper.Created(c, "user berhasil dibuat", newUser,
		"/api/v1/users/"+strconv.Itoa(newUser.ID))
}

// Replace godoc
//
// @Summary      Mengganti data user seluruhnya (PUT)
// @Description  Semua field wajib dikirim. Pemilik data boleh mengubah data sendiri; user lain membutuhkan permission user:update:any.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id    path  int                      true  "ID user"
// @Param        body  body  model.ReplaceUserRequest  true  "Data user yang lengkap"
// @Success      200   {object}  model.WebResponse{data=model.User}
// @Failure      400   {object}  model.ErrorResponse
// @Failure      403   {object}  model.ErrorResponse
// @Failure      404   {object}  model.ErrorResponse
// @Failure      422   {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /users/{id} [put]
func (s *UserService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	if !CanAccessUser(current, id, s.perms, "user:update:any") {
		return helper.Forbidden("tidak berhak mengubah data user lain")
	}

	var req model.ReplaceUserRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	result, err := s.repo.Update(ctx, model.User{
		ID:       id,
		Username: strings.TrimSpace(req.Username),
		Email:    strings.TrimSpace(req.Email),
		IsActive: req.IsActive,
	})
	if err != nil {
		return translateError(err, "user")
	}

	return helper.Success(c, fiber.StatusOK, "user berhasil diganti seluruhnya", result)
}

// Patch godoc
//
// @Summary      Mengubah sebagian data user (PATCH)
// @Description  Hanya field yang dikirim yang berubah; field yang tidak dikirim bernilai nil dan diabaikan.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id    path  int                    true  "ID user"
// @Param        body  body  model.PatchUserRequest  true  "Field yang ingin diubah"
// @Success      200   {object}  model.WebResponse{data=model.User}
// @Failure      400   {object}  model.ErrorResponse  "Tidak ada field yang diubah"
// @Failure      403   {object}  model.ErrorResponse
// @Failure      404   {object}  model.ErrorResponse
// @Failure      422   {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /users/{id} [patch]
func (s *UserService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	if !CanAccessUser(current, id, s.perms, "user:update:any") {
		return helper.Forbidden("tidak berhak mengubah data user lain")
	}

	var req model.PatchUserRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	if IsEmptyPatch(req) {
		return helper.BadRequest("tidak ada field yang diubah")
	}

	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(err, "user")
	}

	updated := ApplyPatch(existing, req)

	result, err := s.repo.Update(ctx, updated)
	if err != nil {
		return translateError(err, "user")
	}

	return helper.Success(c, fiber.StatusOK, "user berhasil diperbarui sebagian", result)
}

// ---------- PATCH /users/:id/role ----------

// AssignRole godoc
//
// @Summary      Mengubah role user
// @Description  Membutuhkan permission role:assign; aturan tambahan diperiksa di service (ValidateAssignRole).
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id    path  int                     true  "ID user"
// @Param        body  body  model.AssignRoleRequest  true  "Role baru"
// @Success      200   {object}  model.WebResponse{data=model.User}
// @Failure      400   {object}  model.ErrorResponse
// @Failure      401   {object}  model.ErrorResponse
// @Failure      403   {object}  model.ErrorResponse
// @Failure      404   {object}  model.ErrorResponse
// @Failure      422   {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /users/{id}/role [patch]
func (s *UserService) AssignRole(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.AssignRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if errs := ValidateAssignRole(current, id, req, s.perms); len(errs) > 0 {
		return helper.Validation(errs)
	}

	result, err := s.repo.UpdateRole(ctx, id, strings.TrimSpace(req.Role))
	if err != nil {
		return translateError(err, "user")
	}

	return helper.Success(c, fiber.StatusOK, "role user berhasil diubah", result)
}

// ---------- DELETE /users/:id ----------

// Delete godoc
//
// @Summary  Menghapus user (admin)
// @Description  Membutuhkan permission user:delete. Menghapus akun sendiri ditolak.
// @Tags     users
// @Produce  json
// @Param    id  path  int  true  "ID user"
// @Success  204  "Berhasil dihapus; tanpa body"
// @Failure  400  {object}  model.ErrorResponse
// @Failure  403  {object}  model.ErrorResponse  "Tidak boleh menghapus akun sendiri"
// @Failure  404  {object}  model.ErrorResponse
// @Security BearerAuth
// @Router   /users/{id} [delete]
func (s *UserService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	if current.UserID == id {
		return helper.Forbidden("tidak boleh menghapus akun sendiri")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return translateError(err, "user")
	}
	return helper.NoContent(c)
}

// translateError mengubah error milik repository menjadi AppError.
func translateError(err error, entity string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound(entity + " tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Conflict("username sudah dipakai")
	default:
		return helper.Internal(err)
	}
}