package helper

import (
	"reflect"
	"testing"
)

func TestCan(t *testing.T) {
	perms := NewPermissionSet(map[string][]string{
		"admin": {"student:list", "student:read:any", "student:create",
			"student:update:any", "student:delete"},
		"staff": {"student:list", "student:read:any", "student:create"},
		"user":  {},
	})

	if !perms.Can("admin", "student:delete") {
		t.Error("admin seharusnya boleh menghapus")
	}
	if !perms.Can("staff", "student:create") {
		t.Error("staff seharusnya boleh menambah")
	}
	if perms.Can("staff", "student:delete") {
		t.Error("staff seharusnya tidak boleh menghapus")
	}
	if perms.Can("user", "student:list") {
		t.Error("user sengaja tidak diberi permission apa pun")
	}
	if perms.Can("rolengawur", "student:list") {
		t.Error("role tak dikenal harus ditolak (fail closed)")
	}
	if perms.Can("admin", "student:lis") {
		t.Error("permission salah ketik harus ditolak (fail closed)")
	}

	var nilSet *PermissionSet
	if nilSet.Can("admin", "student:list") {
		t.Error("receiver nil harus ditolak (fail closed)")
	}
}

func TestPermissionsOf(t *testing.T) {
	perms := NewPermissionSet(map[string][]string{
		"admin": {"student:delete", "student:list", "student:create"},
		"user":  {},
	})

	got := perms.PermissionsOf("admin")
	want := []string{"student:create", "student:delete", "student:list"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("harap %v, dapat %v", want, got)
	}

	if got := perms.PermissionsOf("user"); len(got) != 0 {
		t.Errorf("user seharusnya kosong, dapat %v", got)
	}
	if got := perms.PermissionsOf("rolengawur"); got == nil || len(got) != 0 {
		t.Errorf("role tak dikenal seharusnya slice kosong, dapat %v", got)
	}
}

func TestKnownRoles(t *testing.T) {
	perms := NewPermissionSet(map[string][]string{
		"staff": {},
		"admin": {},
		"user":  {},
	})

	want := []string{"admin", "staff", "user"}
	if got := perms.KnownRoles(); !reflect.DeepEqual(got, want) {
		t.Errorf("harap %v, dapat %v", want, got)
	}
}

func TestIsKnownRole(t *testing.T) {
	perms := NewPermissionSet(map[string][]string{
		"admin": {}, "staff": {}, "user": {},
	})

	if !perms.IsKnownRole("staff") {
		t.Error("staff seharusnya dikenal")
	}
	if perms.IsKnownRole("superadmin") {
		t.Error("superadmin bukan role yang dikenal")
	}
}
