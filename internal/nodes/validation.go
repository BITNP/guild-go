package nodes

import (
	"strings"
	"unicode/utf8"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
)

func normalizeNodeFields(title, description string, tags []string) (string, string, []string, error) {
	title = strings.TrimSpace(title)
	if count := utf8.RuneCountInString(title); count < 1 || count > 200 {
		return "", "", nil, apperr.BadRequest("标题长度必须为 1–200 字符")
	}
	description = strings.TrimSpace(description)
	if utf8.RuneCountInString(description) > 20000 {
		return "", "", nil, apperr.BadRequest("描述不能超过 20000 字符")
	}
	normalizedTags, err := normalizeTags(tags)
	if err != nil {
		return "", "", nil, err
	}
	return title, description, normalizedTags, nil
}

func normalizeTags(tags []string) ([]string, error) {
	result := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if utf8.RuneCountInString(tag) > 32 {
			return nil, apperr.BadRequest("单个标签不能超过 32 字符")
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		result = append(result, tag)
	}
	if len(result) > 10 {
		return nil, apperr.BadRequest("最多只能有 10 个标签")
	}
	return result, nil
}

func normalizeSubjects(subjects []string) []string {
	result := make([]string, 0, len(subjects))
	seen := make(map[string]struct{}, len(subjects))
	for _, subject := range subjects {
		subject = strings.TrimSpace(subject)
		if subject == "" {
			continue
		}
		if _, ok := seen[subject]; ok {
			continue
		}
		seen[subject] = struct{}{}
		result = append(result, subject)
	}
	return result
}

func relationFor(id int64, leaders, members map[int64]struct{}) *string {
	if leaders != nil {
		if _, ok := leaders[id]; ok {
			value := "leader"
			return &value
		}
	}
	if members != nil {
		if _, ok := members[id]; ok {
			value := "member"
			return &value
		}
	}
	return nil
}

func stringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func intersect(a, b map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{})
	for value := range a {
		if _, ok := b[value]; ok {
			result[value] = struct{}{}
		}
	}
	return result
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func nonNilBreadcrumb(values []Breadcrumb) []Breadcrumb {
	if values == nil {
		return []Breadcrumb{}
	}
	return values
}
