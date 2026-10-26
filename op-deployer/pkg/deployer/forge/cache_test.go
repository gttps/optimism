package forge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// tempDir returns a temporary directory without symlinks in its path, as forge would see it.
func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	return dir
}

// newProject creates a project at root with the given directories and the given cache content.
func newProject(t *testing.T, root string, dirs []string, cache map[string]any) string {
	t.Helper()
	for _, dir := range append(dirs, "cache") {
		require.NoError(t, os.MkdirAll(filepath.Join(root, dir), 0o755))
	}
	data, err := json.Marshal(cache)
	require.NoError(t, err)
	path := filepath.Join(root, "cache", "solidity-files-cache.json")
	require.NoError(t, os.WriteFile(path, data, 0o644))
	return path
}

func readCache(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var cache map[string]any
	require.NoError(t, json.Unmarshal(data, &cache))
	return cache
}

func TestRelocateCache(t *testing.T) {
	// The project root has a src/ like the remapped lib/dep/src/, so a short suffix match would
	// pick the wrong root.
	dirs := []string{"src", "lib/dep/src", "lib/other/lib/nested", "interfaces"}

	t.Run("resolves the root from nested targets only", func(t *testing.T) {
		tmp := tempDir(t)
		oldRoot := filepath.Join(tmp, "build")
		root := filepath.Join(tmp, "bundle")
		path := newProject(t, root, dirs, map[string]any{
			"remappings": []string{"dep/=" + oldRoot + "/lib/dep/src/"},
		})

		require.NoError(t, relocateCache(root))
		require.Equal(t, []any{"dep/=" + root + "/lib/dep/src/"}, readCache(t, path)["remappings"])
	})

	t.Run("outvotes a missing target that matches a short suffix", func(t *testing.T) {
		tmp := tempDir(t)
		oldRoot := filepath.Join(tmp, "build")
		root := filepath.Join(tmp, "bundle")
		// lib/gone/src isn't in the bundle, but src/ at its root is, so this target alone would
		// resolve to oldRoot/lib/gone.
		path := newProject(t, root, dirs, map[string]any{
			"remappings": []string{
				"gone/=" + oldRoot + "/lib/gone/src/",
				"dep/=" + oldRoot + "/lib/dep/src/",
				"interfaces/=" + oldRoot + "/interfaces/",
			},
		})

		require.NoError(t, relocateCache(root))
		require.Equal(t, []any{
			"gone/=" + root + "/lib/gone/src/",
			"dep/=" + root + "/lib/dep/src/",
			"interfaces/=" + root + "/interfaces/",
		}, readCache(t, path)["remappings"])
	})

	t.Run("rewrites the old root in remappings", func(t *testing.T) {
		tmp := tempDir(t)
		oldRoot := filepath.Join(tmp, "build", "contracts")
		root := filepath.Join(tmp, "bundle-1")
		files := map[string]any{"src/A.sol": map[string]any{"contentHash": "abc"}}
		path := newProject(t, root, dirs, map[string]any{
			"files": files,
			"paths": map[string]any{"sources": "src"},
			"remappings": []string{
				"dep/=" + oldRoot + "/lib/dep/src/",
				"interfaces/=" + oldRoot + "/interfaces/",
				oldRoot + "/lib/other/:nested/=" + oldRoot + "/lib/other/lib/nested/",
				"lib/other/:nested/=" + oldRoot + "/lib/other/lib/nested/",
				"missing/=" + oldRoot + "/lib/missing/",
			},
		})

		require.NoError(t, relocateCache(root))

		cache := readCache(t, path)
		require.Equal(t, []any{
			"dep/=" + root + "/lib/dep/src/",
			"interfaces/=" + root + "/interfaces/",
			root + "/lib/other/:nested/=" + root + "/lib/other/lib/nested/",
			"lib/other/:nested/=" + root + "/lib/other/lib/nested/",
			"missing/=" + root + "/lib/missing/",
		}, cache["remappings"])
		require.Equal(t, files, cache["files"])
		require.Equal(t, map[string]any{"sources": "src"}, cache["paths"])
	})

	t.Run("resolves a relative or symlinked root to the path forge sees", func(t *testing.T) {
		tmp := tempDir(t)
		oldRoot := filepath.Join(tmp, "build")
		root := filepath.Join(tmp, "bundle")
		remappings := []string{"dep/=" + oldRoot + "/lib/dep/src/"}
		want := []any{"dep/=" + root + "/lib/dep/src/"}
		link := filepath.Join(tmp, "link")
		require.NoError(t, os.MkdirAll(root, 0o755))
		require.NoError(t, os.Symlink(root, link))

		path := newProject(t, root, dirs, map[string]any{"remappings": remappings})
		require.NoError(t, relocateCache(link))
		require.Equal(t, want, readCache(t, path)["remappings"])

		path = newProject(t, root, dirs, map[string]any{"remappings": remappings})
		t.Chdir(tmp)
		require.NoError(t, relocateCache("bundle/"))
		require.Equal(t, want, readCache(t, path)["remappings"])
	})

	t.Run("leaves a cache at its own root unchanged", func(t *testing.T) {
		root := tempDir(t)
		path := newProject(t, root, dirs, map[string]any{
			"remappings": []string{"dep/=" + root + "/lib/dep/src/"},
		})
		before, err := os.ReadFile(path)
		require.NoError(t, err)

		require.NoError(t, relocateCache(root))

		after, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, before, after)
	})

	t.Run("ignores a missing cache", func(t *testing.T) {
		require.NoError(t, relocateCache(tempDir(t)))
	})

	t.Run("ignores a cache without remappings", func(t *testing.T) {
		root := tempDir(t)
		path := newProject(t, root, dirs, map[string]any{"remappings": nil})

		require.NoError(t, relocateCache(root))
		require.Nil(t, readCache(t, path)["remappings"])
	})

	t.Run("fails without writing on remappings that disagree on the root", func(t *testing.T) {
		tmp := tempDir(t)
		root := filepath.Join(tmp, "bundle")
		path := newProject(t, root, dirs, map[string]any{
			"remappings": []string{
				"dep/=" + filepath.Join(tmp, "a") + "/lib/dep/src/",
				"interfaces/=" + filepath.Join(tmp, "b") + "/interfaces/",
			},
		})
		before, err := os.ReadFile(path)
		require.NoError(t, err)

		require.ErrorContains(t, relocateCache(root), "different roots")

		after, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, before, after)
	})

	t.Run("fails on a malformed cache", func(t *testing.T) {
		root := tempDir(t)
		require.NoError(t, os.MkdirAll(filepath.Join(root, "cache"), 0o755))
		path := filepath.Join(root, "cache", "solidity-files-cache.json")
		require.NoError(t, os.WriteFile(path, []byte("{"), 0o644))

		require.Error(t, relocateCache(root))
	})
}
