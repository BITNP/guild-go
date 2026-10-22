package authz

import "strings"

// 从已验证的 ID token claims 中读取 realm 角色。
//
// 本文件不做任何 token 校验：调用方必须先用框架完成签名、issuer、audience、
// 有效期与 nonce 的验证，再把 claims 传进来。

// RealmRolesFromClaims 取出 realm_access.roles。
//
// 字段缺失或类型不符时返回空切片。忽略非字符串和纯空白项；其余字符串
// 保留原文、输入顺序和重复项，不修剪或归一化角色名称。
func RealmRolesFromClaims(claims map[string]any) []string {
	realmAccess, ok := claims["realm_access"].(map[string]any)
	if !ok {
		return nil
	}

	roles := make([]string, 0, 8)
	switch raw := realmAccess["roles"].(type) {
	case []any:
		for _, item := range raw {
			if role, ok := item.(string); ok && strings.TrimSpace(role) != "" {
				roles = append(roles, role)
			}
		}
	case []string:
		for _, role := range raw {
			if strings.TrimSpace(role) != "" {
				roles = append(roles, role)
			}
		}
	default:
		return nil
	}
	return roles
}
