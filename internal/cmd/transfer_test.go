package cmd

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mitchell-wallace/laps/internal/store"
)

func writeTransferFile(t *testing.T, path, prefix string, tasks ...store.Task) {
	t.Helper()
	if err := store.Save(path, &store.File{Version: store.CurrentVersion, Prefix: prefix, Tasks: tasks}); err != nil {
		t.Fatalf("save %s: %v", path, err)
	}
}

func TestTransferAcrossAllQueueCombinationsPreservesTask(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	rootPath := filepath.Join(beadsDir, "laps.json")
	alphaPath, _ := store.ResolveStintFile(beadsDir, "alpha")
	betaPath, _ := store.ResolveStintFile(beadsDir, "beta")
	completed := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	original := store.Task{
		ID:          "laps-abcd",
		Title:       "Preserve me",
		Description: "every field stays intact",
		Assignee:    "SOL",
		IsDone:      true,
		Order:       7340032,
		CreatedAt:   completed.Add(-2 * time.Hour),
		UpdatedAt:   completed.Add(-time.Hour),
		CompletedAt: &completed,
	}
	writeTransferFile(t, rootPath, "", original)
	writeTransferFile(t, alphaPath, "alph")
	writeTransferFile(t, betaPath, "beta")
	expectedFile, err := store.Load(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	expected := expectedFile.Tasks[0]

	for _, tc := range []struct {
		args       []string
		sourcePath string
		targetPath string
	}{
		{[]string{"transfer", "alpha", original.ID, "--root"}, rootPath, alphaPath},
		{[]string{"transfer", "beta", original.ID, "--stint", "alpha"}, alphaPath, betaPath},
		{[]string{"transfer", "root", original.ID, "--stint", "beta"}, betaPath, rootPath},
	} {
		out, errOut, code := runMB(tc.args...)
		if code != 0 {
			t.Fatalf("%v exit %d; stdout=%s stderr=%s", tc.args, code, out, errOut)
		}
		source, err := store.Load(tc.sourcePath)
		if err != nil {
			t.Fatal(err)
		}
		if store.FindTask(source, original.ID) != nil {
			t.Fatalf("%v left task in source", tc.args)
		}
		target, err := store.Load(tc.targetPath)
		if err != nil {
			t.Fatal(err)
		}
		got := store.FindTask(target, original.ID)
		if got == nil || !reflect.DeepEqual(*got, expected) {
			t.Fatalf("%v task changed\n got: %+v\nwant: %+v", tc.args, got, expected)
		}
	}
}

func TestTransferRejectsClaimedTaskWithoutMutation(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	rootPath := filepath.Join(beadsDir, "laps.json")
	alphaPath, _ := store.ResolveStintFile(beadsDir, "alpha")
	task := store.Task{ID: "laps-claim", Title: "Claimed", Order: 1}
	writeTransferFile(t, rootPath, "", task)
	writeTransferFile(t, alphaPath, "alph")
	if err := store.WriteClaim(beadsDir, store.Claim{Lap: task.ID, File: "laps.json", Scope: "root"}); err != nil {
		t.Fatal(err)
	}

	_, errOut, code := runMB("transfer", "alpha", task.ID, "--root")
	if code != 1 || !strings.Contains(errOut, "currently claimed") {
		t.Fatalf("exit=%d stderr=%s", code, errOut)
	}
	source, _ := store.Load(rootPath)
	target, _ := store.Load(alphaPath)
	if store.FindTask(source, task.ID) == nil || store.FindTask(target, task.ID) != nil {
		t.Fatalf("claimed transfer partially mutated source or target")
	}
}

func TestTransferValidatesWholeBatchBeforeMutation(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	rootPath := filepath.Join(beadsDir, "laps.json")
	alphaPath, _ := store.ResolveStintFile(beadsDir, "alpha")
	task := store.Task{ID: "laps-present", Title: "Present", Order: 1}
	writeTransferFile(t, rootPath, "", task)
	writeTransferFile(t, alphaPath, "alph")

	_, errOut, code := runMB("transfer", "alpha", task.ID, "laps-missing", "--root")
	if code != 3 || !strings.Contains(errOut, "laps-missing") {
		t.Fatalf("exit=%d stderr=%s", code, errOut)
	}
	source, _ := store.Load(rootPath)
	target, _ := store.Load(alphaPath)
	if store.FindTask(source, task.ID) == nil || store.FindTask(target, task.ID) != nil {
		t.Fatalf("failed batch partially mutated source or target")
	}
}

func TestTransferredIDOutOfScopeHintUsesCurrentQueue(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	rootPath := filepath.Join(beadsDir, "laps.json")
	alphaPath, _ := store.ResolveStintFile(beadsDir, "alpha")
	task := store.Task{ID: "laps-origin", Title: "Moved owner", Order: 1}
	writeTransferFile(t, rootPath, "", task)
	writeTransferFile(t, alphaPath, "alph")
	if _, errOut, code := runMB("transfer", "alpha", task.ID, "--root"); code != 0 {
		t.Fatalf("transfer exit=%d stderr=%s", code, errOut)
	}

	_, errOut, code := runMB("get", task.ID, "--root")
	if code != 3 || !strings.Contains(errOut, "is in stint alpha") {
		t.Fatalf("out-of-scope hint exit=%d stderr=%s", code, errOut)
	}
}

func TestTransferRejectsArchivedSourceAndTarget(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	rootPath := filepath.Join(beadsDir, "laps.json")
	writeTransferFile(t, rootPath, "", store.Task{ID: "laps-one", Title: "One"})
	archivedPath, _ := store.ResolveArchivedStintFile(beadsDir, "old")
	writeTransferFile(t, archivedPath, "oldx", store.Task{ID: "oldx-one", Title: "Old"})

	_, errOut, code := runMB("transfer", "old", "laps-one", "--root")
	if code != 3 || !strings.Contains(errOut, "target stint old is archived") {
		t.Fatalf("archived target exit=%d stderr=%s", code, errOut)
	}
	_, errOut, code = runMB("transfer", "root", "oldx-one", "--stint", "old")
	if code != 3 || !strings.Contains(errOut, "source stint old is archived") {
		t.Fatalf("archived source exit=%d stderr=%s", code, errOut)
	}
}

func TestTransferLogsFirstClassEvent(t *testing.T) {
	beadsDir, cleanup := setupTempRepo(t)
	defer cleanup()
	rootPath := filepath.Join(beadsDir, "laps.json")
	alphaPath, _ := store.ResolveStintFile(beadsDir, "alpha")
	task := store.Task{ID: "laps-event", Title: "Logged", Assignee: "SOL", Order: 1}
	writeTransferFile(t, rootPath, "", task)
	writeTransferFile(t, alphaPath, "alph")

	_, errOut, code := runMB("transfer", "alpha", task.ID, "--root")
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errOut)
	}
	lines := readEventLog(t, beadsDir)
	if len(lines) != 1 {
		t.Fatalf("event lines=%d, want 1", len(lines))
	}
	event := lines[0]
	if event["event"] != "transferred" || event["cmd"] != "transfer" || event["lap"] != task.ID || event["scope"] != "alpha" || event["file"] != "stints/alpha.laps.json" {
		t.Fatalf("transfer event = %#v", event)
	}
	detail, _ := event["detail"].(map[string]interface{})
	if detail["from"] != "root" || detail["to"] != "alpha" {
		t.Fatalf("transfer detail = %#v", detail)
	}
}
