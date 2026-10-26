package forge

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
)

// relocateCache rewrites the project root recorded in the forge compile cache under root to root,
// so that forge reuses a cache built in another directory, such as one packed into an artifacts
// tarball. Forge compares the cached remappings, which hold absolute paths, with the ones it
// resolves for the current root, and discards the whole cache if they differ.
func relocateCache(root string) error {
	// TODO(foundry-rs/foundry#17829): Remove once forge caches remappings relative to the root.
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	// Forge resolves the root from its working directory, which has no symlinks.
	if root, err = filepath.EvalSymlinks(root); err != nil {
		return err
	}
	path := filepath.Join(root, "cache", "solidity-files-cache.json")
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	var cache map[string]json.RawMessage
	if err := json.Unmarshal(data, &cache); err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}
	var remappings []string
	if raw, ok := cache["remappings"]; ok {
		if err := json.Unmarshal(raw, &remappings); err != nil {
			return fmt.Errorf("failed to parse cached remappings: %w", err)
		}
	}

	oldRoot, err := cachedRoot(root, remappings)
	if err != nil {
		return err
	}
	if oldRoot == "" || oldRoot == root {
		return nil
	}

	oldPrefix := oldRoot + string(filepath.Separator)
	newPrefix := root + string(filepath.Separator)
	for i, remapping := range remappings {
		// A remapping is `[context:]name=path`, and forge stores context and path as absolute paths.
		lhs, target, ok := strings.Cut(remapping, "=")
		if !ok {
			continue
		}
		if rest, ok := strings.CutPrefix(lhs, oldPrefix); ok {
			lhs = newPrefix + rest
		}
		if rest, ok := strings.CutPrefix(target, oldPrefix); ok {
			target = newPrefix + rest
		}
		remappings[i] = lhs + "=" + target
	}
	if cache["remappings"], err = json.Marshal(remappings); err != nil {
		return err
	}
	return jsonutil.WriteJSON(cache, ioutil.ToAtomicFile(path, 0o644))
}

// cachedRoot returns the project root that the absolute remapping targets were resolved against.
// A target's candidate is its shallowest ancestor whose path to the target also exists under root:
// the longest matching suffix, so that `src` at the project root can't match `lib/x/src`. A target
// that's missing under root can still match a short suffix by chance, so the most common candidate
// wins. It returns "" if no target resolves, and an error on a tie.
func cachedRoot(root string, remappings []string) (string, error) {
	votes := make(map[string]int)
	for _, remapping := range remappings {
		_, target, ok := strings.Cut(remapping, "=")
		if !ok || !filepath.IsAbs(target) {
			continue
		}
		target = filepath.Clean(target)
		candidate := ""
		for dir := filepath.Dir(target); ; dir = filepath.Dir(dir) {
			if rel, err := filepath.Rel(dir, target); err == nil {
				if _, err := os.Stat(filepath.Join(root, rel)); err == nil {
					candidate = dir
				}
			}
			if dir == filepath.Dir(dir) {
				break
			}
		}
		if candidate != "" {
			votes[candidate]++
		}
	}
	var found string
	tied := false
	for candidate, n := range votes {
		switch {
		case n > votes[found]:
			found, tied = candidate, false
		case n == votes[found]:
			tied = true
		}
	}
	if tied {
		return "", fmt.Errorf("cached remappings resolve to different roots: %v", votes)
	}
	return found, nil
}
