package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchell-wallace/laps/internal/store"
)

type resolverFailureKind string

const (
	resolverFailureMissingChild  resolverFailureKind = "missing child file"
	resolverFailureMalformedRef  resolverFailureKind = "malformed stint ref"
	resolverFailureMalformedFile resolverFailureKind = "malformed child file"
	resolverFailureCycle         resolverFailureKind = "cycle"
)

type resolverError struct {
	Kind resolverFailureKind
	Ref  string
	Path string
	Err  error
}

func (e *resolverError) Error() string {
	switch e.Kind {
	case resolverFailureMissingChild:
		return fmt.Sprintf("resolution failed: missing child file for stint %q: %s", e.Ref, e.Path)
	case resolverFailureMalformedRef:
		if e.Err != nil {
			return fmt.Sprintf("resolution failed: malformed stint ref %q: %v", e.Ref, e.Err)
		}
		return "resolution failed: malformed stint ref"
	case resolverFailureMalformedFile:
		return fmt.Sprintf("resolution failed: malformed child file for stint %q at %s: %v", e.Ref, e.Path, e.Err)
	case resolverFailureCycle:
		return fmt.Sprintf("resolution failed: cycle detected at stint %q", e.Ref)
	default:
		if e.Err != nil {
			return fmt.Sprintf("resolution failed: %v", e.Err)
		}
		return "resolution failed"
	}
}

func (e *resolverError) Unwrap() error {
	return e.Err
}

type activeContext struct {
	Path  string
	Scope string
	File  *store.File
	Head  *store.Task
}

func resolveActiveContext(rootPath, repoRoot, beadsDir string, rootFile *store.File) (*activeContext, error) {
	scope := "root"
	if name, ok := store.ActiveStintNameForPath(beadsDir, rootPath); ok {
		scope = name
	}
	ctx := &activeContext{
		Path:  rootPath,
		Scope: scope,
		File:  rootFile,
		Head:  firstTodo(rootFile),
	}
	visited := make(map[string]struct{})

	for ctx.Head != nil && ctx.Head.Kind == store.KindStint {
		ref := ctx.Head.Ref
		childPath, err := store.ResolveStintFile(beadsDir, ref)
		if err != nil {
			return nil, &resolverError{Kind: resolverFailureMalformedRef, Ref: ref, Err: err}
		}
		childIdentity, err := filepath.Abs(childPath)
		if err != nil {
			childIdentity = filepath.Clean(childPath)
		}
		if _, seen := visited[childIdentity]; seen {
			return nil, &resolverError{Kind: resolverFailureCycle, Ref: ref, Path: childPath}
		}
		visited[childIdentity] = struct{}{}

		if _, err := os.Stat(childPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, &resolverError{Kind: resolverFailureMissingChild, Ref: ref, Path: childPath}
			}
			return nil, &resolverError{Kind: resolverFailureMalformedFile, Ref: ref, Path: childPath, Err: err}
		}

		childFile, err := loadExistingFile(childPath, repoRoot, beadsDir)
		if err != nil {
			return nil, &resolverError{Kind: resolverFailureMalformedFile, Ref: ref, Path: childPath, Err: err}
		}
		ctx = &activeContext{
			Path:  childPath,
			Scope: appendScope(ctx.Scope, ref),
			File:  childFile,
			Head:  firstTodo(childFile),
		}
	}

	return ctx, nil
}

func firstTodo(file *store.File) *store.Task {
	for i := range file.Tasks {
		if !file.Tasks[i].IsDone {
			return &file.Tasks[i]
		}
	}
	return nil
}

func appendScope(parent, ref string) string {
	if parent == "" || parent == "root" {
		return ref
	}
	return parent + "/" + ref
}

func loadExistingFile(path, repoRoot, beadsDir string) (*store.File, error) {
	data, err := store.Load(path)
	if err != nil {
		return nil, err
	}
	if name, ok := store.ActiveStintNameForPath(beadsDir, path); ok && data.Prefix == "" {
		if len(data.Tasks) > 0 {
			return nil, fmt.Errorf("file is a stint file but is missing prefix metadata; recreate it with `laps stints new %s` or add a 4-character prefix", name)
		}
		prefix, err := store.AllocateStintPrefix(beadsDir, repoRoot, name)
		if err != nil {
			return nil, err
		}
		data.Prefix = prefix
		if err := store.Save(path, data); err != nil {
			return nil, err
		}
	}
	if data.Version > store.CurrentVersion {
		return nil, fmt.Errorf("file %s was written by a newer version of laps (schema version %d); please update laps", path, data.Version)
	}
	if store.Migrate(data) {
		if err := store.Save(path, data); err != nil {
			return nil, err
		}
	}
	store.Normalize(data)
	return data, nil
}

func fileNameForClaim(beadsDir, path string) string {
	if rel, err := filepath.Rel(beadsDir, path); err == nil && rel != "." && !filepath.IsAbs(rel) && !isParentRelative(rel) {
		return filepath.ToSlash(rel)
	}
	return path
}

func isParentRelative(path string) bool {
	return path == ".." || len(path) > 3 && path[:3] == "../"
}

func pathForClaimFile(beadsDir, claimFile string) string {
	if claimFile == "" {
		claimFile = store.ResolveFile(fileFlag)
	}
	if filepath.IsAbs(claimFile) {
		return claimFile
	}
	return filepath.Join(beadsDir, filepath.FromSlash(claimFile))
}
