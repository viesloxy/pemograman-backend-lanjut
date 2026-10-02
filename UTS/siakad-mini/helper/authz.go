package helper

import "sort"

// PermissionSet menyimpan pemetaan role -> permission di memori.
// Pencarian hak menjadi satu langkah, dan sifatnya fail closed:
// role yang tidak dikenal maupun permission yang salah ketik
// selalu menjawab false.
type PermissionSet struct {
	m map[string]map[string]struct{}
}

func NewPermissionSet(mapping map[string][]string) *PermissionSet {
	m := make(map[string]map[string]struct{}, len(mapping))
	for role, perms := range mapping {
		set := make(map[string]struct{}, len(perms))
		for _, p := range perms {
			set[p] = struct{}{}
		}
		m[role] = set
	}
	return &PermissionSet{m: m}
}

// Can menjawab apakah sebuah role memegang sebuah permission.
// Receiver nil, role tak dikenal, dan permission tak dikenal
// semuanya dijawab false: fail closed.
func (p *PermissionSet) Can(role, permission string) bool {
	if p == nil {
		return false
	}
	perms, ok := p.m[role]
	if !ok {
		return false
	}
	_, ok = perms[permission]
	return ok
}

func (p *PermissionSet) KnownRoles() []string {
	if p == nil {
		return nil
	}
	roles := make([]string, 0, len(p.m))
	for role := range p.m {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return roles
}

// SIAKADPermissions adalah kebijakan akses SIAKAD Mini.
// Sufiks :any berarti data milik orang lain; hak atas data sendiri
// tidak membutuhkan permission karena mengalir dari kepemilikan,
// dan diperiksa di layer service.
func SIAKADPermissions() *PermissionSet {
	return NewPermissionSet(map[string][]string{
		"admin": {
			"student:list",
			"student:create",
			"student:update:any",
			"student:delete",
		},
		// Mahasiswa tidak memegang permission data mahasiswa; haknya atas
		// KRS sendiri berupa permission khusus berikut. Pemeriksaan
		// kepemilikan per baris tetap berada di layer service.
		"mahasiswa": {
			"enrollment:create",
			"enrollment:delete",
		},
	})
}
