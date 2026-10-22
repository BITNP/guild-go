package authz

import "strings"

// SystemAdminRole is the only realm role that grants system administration.
const SystemAdminRole = "permission-system-admin"

// Department identifies one of the departments defined by the role contract.
// Its identity does not depend on its display label.
type Department string

const (
	DepartmentTech    Department = "techdept"
	DepartmentClinic  Department = "clinic"
	DepartmentMedia   Department = "digimedia"
	DepartmentTechsvc Department = "techsvc"
)

var departmentOrder = [...]Department{
	DepartmentTech, DepartmentClinic, DepartmentMedia, DepartmentTechsvc,
}

// RoleKey returns the department's key in permission role names.
func (department Department) RoleKey() string {
	return string(department)
}

// Label returns the department's display label, or an empty string for an unknown department.
func (department Department) Label() string {
	switch department {
	case DepartmentTech:
		return "技术部"
	case DepartmentClinic:
		return "电脑诊所"
	case DepartmentMedia:
		return "数字媒体中心"
	case DepartmentTechsvc:
		return "技术保障中心"
	default:
		return ""
	}
}

// Departments returns an independent copy of the departments in canonical order.
func Departments() []Department {
	return append([]Department(nil), departmentOrder[:]...)
}

// AccessRole returns a department's access role name.
// Constructing a role for an unknown department does not make it recognized.
func AccessRole(department Department) string {
	return "permission-" + department.RoleKey() + "-access"
}

// AdminRole returns a department's administration role name.
// Constructing a role for an unknown department does not make it recognized.
func AdminRole(department Department) string {
	return "permission-" + department.RoleKey() + "-admin"
}

// RecognizedRoles returns an independent list of recognized permission roles.
// The system administrator role comes first, followed by access and admin roles
// for each department in canonical order. Position roles are not included.
func RecognizedRoles() []string {
	roles := make([]string, 0, 1+2*len(departmentOrder))
	roles = append(roles, SystemAdminRole)
	for _, department := range departmentOrder {
		roles = append(roles, AccessRole(department), AdminRole(department))
	}
	return roles
}

var recognizedPermissionRoles = func() map[string]struct{} {
	index := make(map[string]struct{})
	for _, role := range RecognizedRoles() {
		index[role] = struct{}{}
	}
	return index
}()

// PermissionRoles returns recognized permission roles in first-occurrence order,
// with duplicates removed. Matching is exact; whitespace is not normalized.
func PermissionRoles(roles []string) []string {
	result := make([]string, 0, len(roles))
	seen := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		if _, ok := recognizedPermissionRoles[role]; !ok {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		result = append(result, role)
	}
	return result
}

// UnrecognizedPermissionRoles returns unrecognized permission-* roles in
// first-occurrence order, with duplicates removed. These roles grant no access.
// The bare prefix "permission-" is also reported.
func UnrecognizedPermissionRoles(roles []string) []string {
	result := make([]string, 0, len(roles))
	seen := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		if !strings.HasPrefix(role, "permission-") {
			continue
		}
		if _, ok := recognizedPermissionRoles[role]; ok {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		result = append(result, role)
	}
	return result
}
