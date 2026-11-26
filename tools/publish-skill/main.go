// publish-skill packages a kagent Agent Skill directory as a scratch OCI
// image (one tar layer, no base image) and pushes it to a registry, printing
// the resulting digest-pinned reference - the only form kagent's
// AgentTemplate.spec.skills[].source.oci accepts. Mirrors the exact
// publishSkillImage recipe kagent's own e2e suite uses
// (go/core/test/e2e/skills_test.go in enterprise-kagent).
package main

import (
	"archive/tar"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

func main() {
	dir := flag.String("dir", "", "skill source directory to package (required)")
	ref := flag.String("ref", "", "destination image reference, e.g. registry/repo/agora-skills/brand-check:v1.0.0 (required)")
	flag.Parse()
	if *dir == "" || *ref == "" {
		fmt.Fprintln(os.Stderr, "usage: publish-skill -dir <path> -ref <image:tag>")
		os.Exit(2)
	}

	tagRef, err := name.ParseReference(*ref)
	if err != nil {
		log.Fatalf("parse ref %q: %v", *ref, err)
	}

	archive, err := tarDirectory(*dir)
	if err != nil {
		log.Fatalf("tar %q: %v", *dir, err)
	}
	layer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(archive)), nil
	})
	if err != nil {
		log.Fatalf("build layer: %v", err)
	}
	image, err := mutate.AppendLayers(empty.Image, layer)
	if err != nil {
		log.Fatalf("append layer: %v", err)
	}
	if err := remote.Write(tagRef, image, remote.WithAuthFromKeychain(authn.DefaultKeychain)); err != nil {
		log.Fatalf("push %q: %v", tagRef, err)
	}
	digest, err := image.Digest()
	if err != nil {
		log.Fatalf("compute digest: %v", err)
	}
	fmt.Println(tagRef.Context().Digest(digest.String()).Name())
}

// tarDirectory archives dir's regular files, relative to dir, preserving
// each file's executable bit - kagent mounts the skill tree verbatim, and
// probe/run scripts must stay executable.
func tarDirectory(dir string) ([]byte, error) {
	var buf bytes.Buffer
	w := tar.NewWriter(&buf)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		mode := int64(0o644)
		if info.Mode()&0o111 != 0 {
			mode = 0o755
		}
		if err := w.WriteHeader(&tar.Header{Name: rel, Size: int64(len(body)), Mode: mode, Typeflag: tar.TypeReg}); err != nil {
			return err
		}
		_, err = w.Write(body)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
