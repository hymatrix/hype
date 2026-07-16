package cli

import goarSchema "github.com/permadao/goar/schema"

func hasTag(tags []goarSchema.Tag, name, value string) bool {
	for _, tag := range tags {
		if tag.Name == name && tag.Value == value {
			return true
		}
	}
	return false
}
