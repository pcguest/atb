// SPDX-License-Identifier: MIT
package compliancepack

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

// AddArtifact returns a new pack containing file with regenerated manifest and
// checksums. Existing pack and file byte slices are not mutated.
func AddArtifact(pack Pack, file File) (Pack, error) {
	if file.Name == "" || file.Name == "MANIFEST.json" || file.Name == "SHA256SUMS" {
		return Pack{}, fmt.Errorf("compliance pack: invalid added artifact name %q", file.Name)
	}
	files := make([]File, 0, len(pack.Files)+1)
	for _, existing := range pack.Files {
		if existing.Name == "MANIFEST.json" || existing.Name == "SHA256SUMS" {
			continue
		}
		if existing.Name == file.Name {
			return Pack{}, fmt.Errorf("compliance pack: artifact %q already exists", file.Name)
		}
		files = append(files, File{
			Name:    existing.Name,
			Content: append([]byte(nil), existing.Content...),
		})
	}
	files = append(files, File{Name: file.Name, Content: append([]byte(nil), file.Content...)})
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	manifest := pack.Manifest
	manifest.Files = make([]FileEntry, 0, len(files))
	for _, artifact := range files {
		sum := sha256.Sum256(artifact.Content)
		manifest.Files = append(manifest.Files, FileEntry{
			Name:        artifact.Name,
			SHA256:      hex.EncodeToString(sum[:]),
			SizeBytes:   len(artifact.Content),
			Description: describe(artifact.Name),
		})
	}
	manifestJSON, err := marshalJSON(manifest)
	if err != nil {
		return Pack{}, err
	}
	files = append(files,
		File{Name: "MANIFEST.json", Content: manifestJSON},
		File{Name: "SHA256SUMS", Content: checksumFile(manifest.Files)},
	)
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	return Pack{GeneratedAt: pack.GeneratedAt, Files: files, Manifest: manifest}, nil
}
