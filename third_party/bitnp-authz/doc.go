// Package authz 将已验证的 realm 角色换算成成员身份、管理权限、部门与职务。
//
// 只做纯计算：不联网、不校验 token、不依赖任何框架。调用方先用自己的框架完成
// OIDC 登录与 token 校验，再把验证过的 claims 交给本包。
//
// 典型用法（调用方导入标准库 slices 和本包 authz）：
//
//	roles := authz.RealmRolesFromClaims(verifiedClaims)
//	perms := authz.FromRoles(roles)
//	if !perms.Admitted {
//		// 调用方自行决定错误响应
//	}
//	if perms.SystemAdmin {
//		// 全局权限
//	}
//	if slices.Contains(perms.ManagedDepartments, authz.DepartmentTech) {
//		// 技术部管理权限
//	}
//
// 契约见仓库根目录的 SPEC.md；本包与其一一对应。
package authz
