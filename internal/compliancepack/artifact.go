// SPDX-License-Identifier: MIT
package compliancepack

import (
	"fmt"
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
	return finalizePack(pack.GeneratedAt, pack.Manifest, files)
}
