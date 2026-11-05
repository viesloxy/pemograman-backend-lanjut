package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RoleRepository membaca pemetaan role ke permission dari database.
type RoleRepository interface {
	LoadPermissions(ctx context.Context) (map[string][]string, error)
}

type rolePostgresRepository struct{ pool *pgxpool.Pool }

func NewRoleRepository(pool *pgxpool.Pool) RoleRepository {
	return &rolePostgresRepository{pool: pool}
}

// LoadPermissions mengambil SELURUH pasangan role dan permission sekaligus.
//
// LEFT JOIN dipakai dengan sengaja: role yang belum punya permission apa pun
// tetap harus muncul agar sistem tahu role tersebut memang dikenal.
func (r *rolePostgresRepository) LoadPermissions(
	ctx context.Context,
) (map[string][]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.name, COALESCE(rp.permission_name, '')
		 FROM roles r
		 LEFT JOIN role_permissions rp ON rp.role_name = r.name
		 ORDER BY r.name, rp.permission_name`)
	if err != nil {
		return nil, fmt.Errorf("mengambil permission: %w", err)
	}
	defer rows.Close()

	result := map[string][]string{}
	for rows.Next() {
		var role, permission string
		if err := rows.Scan(&role, &permission); err != nil {
			return nil, fmt.Errorf("membaca row permission: %w", err)
		}
		if _, ok := result[role]; !ok {
			result[role] = []string{}
		}
		if permission != "" {
			result[role] = append(result[role], permission)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca hasil query permission: %w", err)
	}
	return result, nil
}