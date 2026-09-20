package repository

import (
	"context"
	"os"
	"sort"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func TestLoadPermissions(t *testing.T) {
	_ = godotenv.Load("../../.env")

	dsn := "postgres://" + os.Getenv("DB_USER") + ":" + os.Getenv("DB_PASSWORD") +
		"@" + os.Getenv("DB_HOST") + ":" + os.Getenv("DB_PORT") + "/" + os.Getenv("DB_NAME") +
		"?sslmode=" + os.Getenv("DB_SSLMODE")

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	repo := NewRoleRepository(pool)
	raw, err := repo.LoadPermissions(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(raw) != 3 {
		t.Fatalf("harusnya 3 role, dapat %d: %v", len(raw), raw)
	}
	if len(raw["admin"]) != 5 {
		t.Errorf("admin harusnya 5 permission, dapat %v", raw["admin"])
	}
	if len(raw["staff"]) != 3 {
		t.Errorf("staff harusnya 3 permission, dapat %v", raw["staff"])
	}
	if raw["user"] == nil || len(raw["user"]) != 0 {
		t.Errorf("user harusnya ada sebagai role tanpa permission, dapat %v", raw["user"])
	}

	perms := raw["admin"]
	sort.Strings(perms)
	want := []string{"student:create", "student:delete", "student:list",
		"student:read:any", "student:update:any"}
	if !equal(perms, want) {
		t.Errorf("isi permission admin berbeda: %v", perms)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
