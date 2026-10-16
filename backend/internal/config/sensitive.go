package config

import (
	"path"
	"slices"
	"strings"

	"lemmary/backend/internal/aiprovider"
)

// DefaultSensitiveFolder is the consume-folder name a fresh install marks
// sensitive.
const DefaultSensitiveFolder = "sensitive"

// MaxSensitiveFoldersChars bounds the stored list, matching the field's Max.
const MaxSensitiveFoldersChars = 500

// ParseSensitiveFolders splits a comma- or newline-separated list into
// distinct folder names, compared case-insensitively. Separators inside a name
// are dropped: a name is matched against one path segment, never a path.
func ParseSensitiveFolders(raw string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' }) {
		name := strings.Trim(strings.TrimSpace(part), `/\`)
		if name == "" || strings.ContainsAny(name, `/\`) {
			continue
		}
		if !slices.ContainsFunc(out, func(seen string) bool { return strings.EqualFold(seen, name) }) {
			out = append(out, name)
		}
	}
	return out
}

// InSensitiveFolder reports whether a consume-folder path sits under one of the
// folders, at any depth. Only directories count: a file named like a folder is
// not marked by its own name.
func InSensitiveFolder(rel string, folders []string) bool {
	dir := path.Dir(strings.ReplaceAll(rel, `\`, "/"))
	if dir == "." || dir == "/" {
		return false
	}
	for segment := range strings.SplitSeq(dir, "/") {
		if slices.ContainsFunc(folders, func(name string) bool { return strings.EqualFold(name, segment) }) {
			return true
		}
	}
	return false
}

// EmbedsSensitive reports whether sensitive documents may be embedded: only by
// the local embeddings sidecar, which never sends text off the server.
func EmbedsSensitive(cfg Config) bool {
	p := cfg.EmbeddingProvider
	return p != nil && p.SDK == aiprovider.SDKLocalEmbeddings
}
