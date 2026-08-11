package auth

// Permission helpers and PermissionSet.
//
// The Permission type itself is defined in internal/platform/authcontext and
// aliased here. This file keeps the PermissionSet container and conversion
// helpers that the RBAC fallback logic depends on.

// PermissionSet is a helper for checking multiple permissions efficiently.
type PermissionSet map[Permission]struct{}

// NewPermissionSet creates a permission set from a slice of permissions.
func NewPermissionSet(permissions []Permission) PermissionSet {
	set := make(PermissionSet, len(permissions))
	for _, p := range permissions {
		set[p] = struct{}{}
	}
	return set
}

// NewPermissionSetFromStrings creates a permission set from string permissions.
func NewPermissionSetFromStrings(permissions []string) PermissionSet {
	set := make(PermissionSet, len(permissions))
	for _, p := range permissions {
		set[Permission(p)] = struct{}{}
	}
	return set
}

// Contains checks if the set contains a permission.
func (ps PermissionSet) Contains(permission Permission) bool {
	_, exists := ps[permission]
	return exists
}

// ContainsResourceAction checks if the set contains a resource:action permission.
func (ps PermissionSet) ContainsResourceAction(resource, action string) bool {
	return ps.Contains(NewPermission(resource, action))
}

// ContainsAny checks if the set contains any of the given permissions.
func (ps PermissionSet) ContainsAny(permissions ...Permission) bool {
	for _, p := range permissions {
		if ps.Contains(p) {
			return true
		}
	}
	return false
}

// ContainsAll checks if the set contains all of the given permissions.
func (ps PermissionSet) ContainsAll(permissions ...Permission) bool {
	for _, p := range permissions {
		if !ps.Contains(p) {
			return false
		}
	}
	return true
}

// ToSlice converts the permission set to a slice.
func (ps PermissionSet) ToSlice() []Permission {
	result := make([]Permission, 0, len(ps))
	for p := range ps {
		result = append(result, p)
	}
	return result
}

// PermissionsToStrings converts a slice of Permission to a slice of strings.
func PermissionsToStrings(permissions []Permission) []string {
	result := make([]string, len(permissions))
	for i, p := range permissions {
		result[i] = string(p)
	}
	return result
}

// StringsToPermissions converts a slice of strings to a slice of Permission.
func StringsToPermissions(permissions []string) []Permission {
	result := make([]Permission, len(permissions))
	for i, p := range permissions {
		result[i] = Permission(p)
	}
	return result
}

