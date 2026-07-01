package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	Chain []stintDescent
}

type queueState string

const (
	queueStateLap      queueState = "lap"
	queueStateHeld     queueState = "held"
	queueStateEmpty    queueState = "empty"
	queueStateComplete queueState = "complete"
)

type flowResolution struct {
	State queueState
	Ctx   *activeContext
	Held  *heldGate
}

type heldGate struct {
	Stint string
	Ref   *store.Task
	Scope string
	Path  string
}

type stintDescent struct {
	ParentPath string
	ParentFile *store.File
	RefIndex   int
	Ref        *store.Task
	ChildPath  string
	ChildName  string
	ChildHeld  bool
	Scope      string
}

func resolveSelectedContext(path, repoRoot, beadsDir string, file *store.File) (*activeContext, error) {
	if activeScopeSelected() {
		return resolveActiveContext(path, repoRoot, beadsDir, file)
	}
	scope := "root"
	if name, ok := store.ActiveStintNameForPath(beadsDir, path); ok {
		scope = name
	} else if fileFlag != "" {
		scope = fileNameForClaim(beadsDir, path)
	}
	return &activeContext{
		Path:  path,
		Scope: scope,
		File:  file,
		Head:  firstTodo(file),
	}, nil
}

func activeScopeSelected() bool {
	return scopeActive || (fileFlag == "" && !scopeRoot && scopeStint == "")
}

func findScopedTask(ctx *activeContext, id string) *store.Task {
	for i := range ctx.File.Tasks {
		if ctx.File.Tasks[i].ID == id {
			return &ctx.File.Tasks[i]
		}
	}
	return nil
}

func exitIfOutOfScope(beadsDir, repoRoot string, ctx *activeContext, id string) {
	prefix, ok := idPrefix(id)
	if !ok {
		return
	}
	selectedPrefix := store.RepoPrefix(repoRoot)
	if ctx.File.Prefix != "" {
		selectedPrefix = ctx.File.Prefix
	}
	if prefix == selectedPrefix {
		return
	}
	if prefix == store.RepoPrefix(repoRoot) {
		exit(3, "%s is in root - re-run with --root", id)
	}
	owners, err := store.StintPrefixMap(beadsDir)
	if err != nil {
		exit(2, "%v", err)
	}
	if stint, ok := owners[prefix]; ok {
		exit(3, "%s is in stint %s - re-run with -s %s", id, stint, stint)
	}
}

func idPrefix(id string) (string, bool) {
	prefix, _, ok := strings.Cut(id, "-")
	if !ok || prefix == "" {
		return "", false
	}
	return prefix, true
}

func resolveActiveContext(rootPath, repoRoot, beadsDir string, rootFile *store.File) (*activeContext, error) {
	resolved, err := resolveFlowStart(rootPath, repoRoot, beadsDir, rootFile, false)
	if err != nil {
		return nil, err
	}
	return resolved.Ctx, nil
}

func resolveSelectedFlowStart(path, repoRoot, beadsDir string, file *store.File, stopOnHeld bool) (*flowResolution, error) {
	if activeScopeSelected() {
		return resolveFlowStart(path, repoRoot, beadsDir, file, stopOnHeld)
	}
	ctx, err := resolveSelectedContext(path, repoRoot, beadsDir, file)
	if err != nil {
		return nil, err
	}
	if stopOnHeld && ctx.File.Held {
		if name, ok := store.ActiveStintNameForPath(beadsDir, ctx.Path); ok {
			return &flowResolution{
				State: queueStateHeld,
				Ctx:   ctx,
				Held: &heldGate{
					Stint: name,
					Scope: ctx.Scope,
					Path:  ctx.Path,
				},
			}, nil
		}
	}
	return &flowResolution{
		State: queueStateForContext(ctx),
		Ctx:   ctx,
	}, nil
}

func resolveFlowStart(rootPath, repoRoot, beadsDir string, rootFile *store.File, stopOnHeld bool) (*flowResolution, error) {
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
		childScope := appendScope(ctx.Scope, ref)
		if stopOnHeld && childFile.Held {
			return &flowResolution{
				State: queueStateHeld,
				Ctx:   ctx,
				Held: &heldGate{
					Stint: ref,
					Ref:   ctx.Head,
					Scope: childScope,
					Path:  childPath,
				},
			}, nil
		}
		ctx = &activeContext{
			Path:  childPath,
			Scope: childScope,
			File:  childFile,
			Head:  firstTodo(childFile),
			Chain: append(append([]stintDescent(nil), ctx.Chain...), stintDescent{
				ParentPath: ctx.Path,
				ParentFile: ctx.File,
				RefIndex:   taskIndex(ctx.File, ctx.Head),
				Ref:        ctx.Head,
				ChildPath:  childPath,
				ChildName:  ref,
				ChildHeld:  childFile.Held,
				Scope:      childScope,
			}),
		}
	}

	return &flowResolution{
		State: queueStateForContext(ctx),
		Ctx:   ctx,
	}, nil
}

func queueStateForContext(ctx *activeContext) queueState {
	if ctx == nil || ctx.File == nil || len(ctx.File.Tasks) == 0 {
		return queueStateEmpty
	}
	if ctx.Head == nil {
		return queueStateComplete
	}
	return queueStateLap
}

func heldGateForContext(ctx *activeContext) *heldGate {
	if ctx == nil {
		return nil
	}
	for i := range ctx.Chain {
		edge := &ctx.Chain[i]
		if !edge.ChildHeld {
			continue
		}
		return &heldGate{
			Stint: edge.ChildName,
			Ref:   edge.Ref,
			Scope: edge.Scope,
			Path:  edge.ChildPath,
		}
	}
	if ctx.File != nil && ctx.File.Held {
		name := ctx.Scope
		if strings.Contains(name, "/") {
			parts := strings.Split(name, "/")
			name = parts[len(parts)-1]
		}
		return &heldGate{
			Stint: name,
			Scope: ctx.Scope,
			Path:  ctx.Path,
		}
	}
	return nil
}

func warnHeldGate(gate *heldGate) {
	if gate == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "laps: stint %s is held; do not implement laps in it yet.\n", gate.Scope)
}

func exitForQueueState(state queueState) int {
	switch state {
	case queueStateHeld:
		return 10
	case queueStateEmpty:
		return 11
	case queueStateComplete:
		return 12
	default:
		return 0
	}
}

func resolvePhysicalStintChain(beadsDir, repoRoot, targetName string, includeArchived bool) ([]stintDescent, bool, error) {
	rootPath := scopedRootPath(beadsDir)
	rootFile := loadFile(rootPath, repoRoot, beadsDir)
	visited := make(map[string]struct{})
	return findPhysicalStintChain(beadsDir, repoRoot, rootPath, rootFile, "root", targetName, includeArchived, visited)
}

func findPhysicalStintChain(beadsDir, repoRoot, parentPath string, parentFile *store.File, parentScope, targetName string, includeArchived bool, visited map[string]struct{}) ([]stintDescent, bool, error) {
	for i := range parentFile.Tasks {
		ref := &parentFile.Tasks[i]
		if ref.Kind != store.KindStint {
			continue
		}
		childPath, childArchived, err := resolveChainChildPath(beadsDir, ref.Ref, includeArchived)
		if err != nil {
			return nil, false, err
		}
		if childPath == "" {
			continue
		}
		childScope := appendScope(parentScope, ref.Ref)
		edge := stintDescent{
			ParentPath: parentPath,
			ParentFile: parentFile,
			RefIndex:   i,
			Ref:        ref,
			ChildPath:  childPath,
			ChildName:  ref.Ref,
			Scope:      childScope,
		}
		if ref.Ref == targetName {
			return []stintDescent{edge}, true, nil
		}

		childIdentity, err := filepath.Abs(childPath)
		if err != nil {
			childIdentity = filepath.Clean(childPath)
		}
		if _, seen := visited[childIdentity]; seen {
			continue
		}
		visited[childIdentity] = struct{}{}

		childFile, err := loadChainChildFile(childPath, repoRoot, beadsDir, childArchived)
		if err != nil {
			return nil, false, err
		}
		tail, ok, err := findPhysicalStintChain(beadsDir, repoRoot, childPath, childFile, childScope, targetName, includeArchived, visited)
		if err != nil {
			return nil, false, err
		}
		if ok {
			return append([]stintDescent{edge}, tail...), true, nil
		}
	}
	return nil, false, nil
}

func resolveChainChildPath(beadsDir, name string, includeArchived bool) (string, bool, error) {
	activePath, err := store.ResolveStintFile(beadsDir, name)
	if err != nil {
		return "", false, err
	}
	if _, err := os.Stat(activePath); err == nil {
		return activePath, false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, err
	}
	if !includeArchived {
		return "", false, nil
	}
	archivedPath, err := store.ResolveArchivedStintFile(beadsDir, name)
	if err != nil {
		return "", false, err
	}
	if _, err := os.Stat(archivedPath); err == nil {
		return archivedPath, true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, err
	}
	return "", false, nil
}

func loadChainChildFile(path, repoRoot, beadsDir string, archived bool) (*store.File, error) {
	if !archived {
		return loadExistingFile(path, repoRoot, beadsDir)
	}
	file, err := store.Load(path)
	if err != nil {
		return nil, err
	}
	if file.Version > store.CurrentVersion {
		return nil, fmt.Errorf("file %s was written by a newer version of laps (schema version %d); please update laps", path, file.Version)
	}
	if store.Migrate(file) {
		if err := store.Save(path, file); err != nil {
			return nil, err
		}
	}
	store.Normalize(file)
	return file, nil
}

func taskIndex(file *store.File, task *store.Task) int {
	if file == nil || task == nil {
		return -1
	}
	for i := range file.Tasks {
		if &file.Tasks[i] == task {
			return i
		}
	}
	return -1
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

func normalizeClaimScope(claim store.Claim) string {
	if claim.Scope != "" {
		return claim.Scope
	}
	return claimScopeFromFile(claim.File)
}

func claimScopeFromFile(claimFile string) string {
	if claimFile == "" || claimFile == store.ResolveFile("") {
		return "root"
	}
	parts := strings.Split(filepath.ToSlash(claimFile), "/")
	if len(parts) == 2 && parts[0] == "stints" && strings.HasSuffix(parts[1], ".laps.json") {
		return strings.TrimSuffix(parts[1], ".laps.json")
	}
	return claimFile
}

func isParentRelative(path string) bool {
	return path == ".." || len(path) > 3 && path[:3] == "../"
}

func pathForClaim(beadsDir string, claim store.Claim) (string, error) {
	scope := normalizeClaimScope(claim)
	switch {
	case scope == "" || scope == "root":
		return filepath.Join(beadsDir, store.ResolveFile("")), nil
	case strings.Contains(scope, "/"):
		parts := strings.Split(scope, "/")
		name := parts[len(parts)-1]
		return store.ResolveStintFile(beadsDir, name)
	case claim.Scope != "":
		return store.ResolveStintFile(beadsDir, scope)
	default:
		return pathForClaimFile(beadsDir, claim.File), nil
	}
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
