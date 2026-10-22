package authz

// SitePermissions 是一个用户的授权事实。
//
// 只有显式出现在 token 里的已识别 permission-* 角色授予权限；Positions 仅用于展示。
type SitePermissions struct {
	Admitted           bool
	SystemAdmin        bool
	Manager            bool
	AccessDepartments  []Department // 已按规范顺序排列
	ManagedDepartments []Department // 已按规范顺序排列
	Positions          []PositionView
}

// FromRoles 把 realm 角色解析成权限与职务。
//
// 角色精确匹配，不修剪空白。部门管理和系统管理角色均不隐含部门访问权限；
// 若应用需要访问权限，token 必须明确包含配对的 access 角色。
func FromRoles(roles []string) SitePermissions {
	roleSet := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		roleSet[role] = struct{}{}
	}

	_, systemAdmin := roleSet[SystemAdminRole]
	permissions := SitePermissions{
		SystemAdmin:        systemAdmin,
		AccessDepartments:  make([]Department, 0, len(departmentOrder)),
		ManagedDepartments: make([]Department, 0, len(departmentOrder)),
		// 保留原始输入顺序，确保同级职务保留先出现的角色。
		Positions: PositionsFromRoles(roles),
	}
	for _, department := range departmentOrder {
		if _, ok := roleSet[AccessRole(department)]; ok {
			permissions.AccessDepartments = append(permissions.AccessDepartments, department)
		}
		if _, ok := roleSet[AdminRole(department)]; ok {
			permissions.ManagedDepartments = append(permissions.ManagedDepartments, department)
		}
	}
	permissions.Manager = systemAdmin || len(permissions.ManagedDepartments) > 0
	permissions.Admitted = permissions.Manager || len(permissions.AccessDepartments) > 0
	return permissions
}

// FromClaims 从已验证的 ID token claims 直接推导权限。
//
// 等价于 FromRoles(RealmRolesFromClaims(claims))；claims 里没有
// realm_access.roles 时返回未准入的结果（Admitted 为 false）。
func FromClaims(claims map[string]any) SitePermissions {
	return FromRoles(RealmRolesFromClaims(claims))
}
