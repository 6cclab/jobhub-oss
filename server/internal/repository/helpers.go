package repository

import "strings"

// joinTags joins a slice of tags into the flat comma-separated TEXT format
// used by the tags columns.
func joinTags(tags []string) string {
	return strings.Join(tags, ",")
}

// splitTags splits a flat comma-separated tags TEXT column back into a slice.
// Returns nil for an empty string.
func splitTags(tags string) []string {
	if tags == "" {
		return nil
	}
	return strings.Split(tags, ",")
}
