package authz

// 职务角色只描述供展示的行政职务，不授予任何权限。

// PositionScope 表示职务属于协会层面还是部门层面。
type PositionScope string

const (
	ScopeAssociation PositionScope = "association"
	ScopeDepartment  PositionScope = "department"
)

// Position 是一个可由 Keycloak 角色映射的行政职务。
type Position struct {
	RoleName      string
	Label         string
	Scope         PositionScope
	DepartmentKey string // 协会层面为空
	Rank          int
}

// positionCatalog 是私有且权威的职务目录。
var positionCatalog = [...]Position{
	{RoleName: "position-association-chair", Label: "协会主席", Scope: ScopeAssociation, Rank: 100},
	{RoleName: "position-association-vice-chair-techdept", Label: "协会副主席（技术部分管）", Scope: ScopeAssociation, Rank: 92},
	{RoleName: "position-association-vice-chair-clinic", Label: "协会副主席（电脑诊所分管）", Scope: ScopeAssociation, Rank: 92},
	{RoleName: "position-association-vice-chair-digimedia", Label: "协会副主席（数字媒体中心分管）", Scope: ScopeAssociation, Rank: 92},

	{RoleName: "position-techdept-head", Label: "技术部部长", Scope: ScopeDepartment, DepartmentKey: "techdept", Rank: 80},
	{RoleName: "position-techdept-deputy-head", Label: "技术部副部长", Scope: ScopeDepartment, DepartmentKey: "techdept", Rank: 70},
	{RoleName: "position-techdept-advisor", Label: "技术部顾问", Scope: ScopeDepartment, DepartmentKey: "techdept", Rank: 60},
	{RoleName: "position-techdept-member", Label: "技术部成员", Scope: ScopeDepartment, DepartmentKey: "techdept", Rank: 10},

	{RoleName: "position-clinic-head", Label: "电脑诊所部长", Scope: ScopeDepartment, DepartmentKey: "clinic", Rank: 80},
	{RoleName: "position-clinic-deputy-head", Label: "电脑诊所副部长", Scope: ScopeDepartment, DepartmentKey: "clinic", Rank: 70},
	{RoleName: "position-clinic-advisor", Label: "电脑诊所顾问", Scope: ScopeDepartment, DepartmentKey: "clinic", Rank: 60},
	{RoleName: "position-clinic-member", Label: "电脑诊所成员", Scope: ScopeDepartment, DepartmentKey: "clinic", Rank: 10},

	{RoleName: "position-digimedia-head", Label: "数字媒体中心部长", Scope: ScopeDepartment, DepartmentKey: "digimedia", Rank: 80},
	{RoleName: "position-digimedia-deputy-head", Label: "数字媒体中心副部长", Scope: ScopeDepartment, DepartmentKey: "digimedia", Rank: 70},
	{RoleName: "position-digimedia-advisor", Label: "数字媒体中心顾问", Scope: ScopeDepartment, DepartmentKey: "digimedia", Rank: 60},
	{RoleName: "position-digimedia-member", Label: "数字媒体中心成员", Scope: ScopeDepartment, DepartmentKey: "digimedia", Rank: 10},

	{RoleName: "position-techsvc-director", Label: "技术保障中心主任", Scope: ScopeDepartment, DepartmentKey: "techsvc", Rank: 80},
	{RoleName: "position-techsvc-deputy-director", Label: "技术保障中心副主任", Scope: ScopeDepartment, DepartmentKey: "techsvc", Rank: 70},
	{RoleName: "position-techsvc-member", Label: "技术保障中心成员", Scope: ScopeDepartment, DepartmentKey: "techsvc", Rank: 10},
}

// Positions 返回权威职务目录的独立副本。
func Positions() []Position {
	return append([]Position(nil), positionCatalog[:]...)
}

var positionByRole = func() map[string]Position {
	index := make(map[string]Position, len(positionCatalog))
	for _, position := range positionCatalog {
		index[position.RoleName] = position
	}
	return index
}()

// PositionForRole 精确查找已识别的职务角色，返回独立值。
// 未知角色返回 Position 零值和 false。
func PositionForRole(role string) (Position, bool) {
	position, ok := positionByRole[role]
	return position, ok
}

var departmentByKey = func() map[string]Department {
	index := make(map[string]Department, len(departmentOrder))
	for _, department := range departmentOrder {
		index[department.RoleKey()] = department
	}
	return index
}()

// PositionView 是某个用户的一个职务，可直接展示。
type PositionView struct {
	Code       string
	Label      string
	Scope      PositionScope
	Department *Department // 协会层面为 nil
}

// PositionsFromRoles 解析并去重角色中的职务。
//
// 协会层面只保留级别最高的职务，每个部门只保留级别最高的职务；同 rank 时保留
// 输入中先出现的职务。返回顺序为协会优先，然后按 TECH、CLINIC、MEDIA、TECHSVC 排列。
// 未识别的角色被忽略。
func PositionsFromRoles(roles []string) []PositionView {
	var association *Position
	byDepartment := make(map[string]Position, len(departmentOrder))

	for _, role := range roles {
		position, ok := PositionForRole(role)
		if !ok {
			continue
		}
		if position.Scope == ScopeAssociation {
			if association == nil || position.Rank > association.Rank {
				candidate := position
				association = &candidate
			}
			continue
		}
		current, ok := byDepartment[position.DepartmentKey]
		if !ok || position.Rank > current.Rank {
			byDepartment[position.DepartmentKey] = position
		}
	}

	views := make([]PositionView, 0, len(byDepartment)+1)
	if association != nil {
		views = append(views, positionView(*association))
	}
	for _, department := range departmentOrder {
		if position, ok := byDepartment[department.RoleKey()]; ok {
			views = append(views, positionView(position))
		}
	}
	return views
}

func positionView(position Position) PositionView {
	view := PositionView{
		Code:  position.RoleName,
		Label: position.Label,
		Scope: position.Scope,
	}
	if position.Scope == ScopeDepartment {
		if department, ok := departmentByKey[position.DepartmentKey]; ok {
			candidate := department
			view.Department = &candidate
		}
	}
	return view
}
