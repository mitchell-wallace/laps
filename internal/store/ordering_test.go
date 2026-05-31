package store

import (
	"errors"
	"testing"
	"time"
)

// ids returns the task ids of f in current array order.
func ids(f *File) []string {
	out := make([]string, len(f.Tasks))
	for i, t := range f.Tasks {
		out[i] = t.ID
	}
	return out
}

func todo(id string, order int) Task { return Task{ID: id, Order: order} }

func done(id string, completed time.Time) Task {
	c := completed
	return Task{ID: id, IsDone: true, CompletedAt: &c}
}

func TestMigrateAssignsOrderInArrayOrder(t *testing.T) {
	f := &File{Version: 1, Tasks: []Task{
		{ID: "a"}, {ID: "b"}, {ID: "c"},
	}}
	if !Migrate(f) {
		t.Fatal("expected Migrate to report a change")
	}
	if f.Version != CurrentVersion {
		t.Fatalf("version = %d, want %d", f.Version, CurrentVersion)
	}
	want := []int{orderStep, 2 * orderStep, 3 * orderStep}
	for i, exp := range want {
		if f.Tasks[i].Order != exp {
			t.Errorf("task %d order = %d, want %d", i, f.Tasks[i].Order, exp)
		}
	}
	// Idempotent: a second migrate is a no-op.
	if Migrate(f) {
		t.Error("expected second Migrate to be a no-op")
	}
}

func TestNormalizeDoneAboveTodoOldestFirst(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	f := &File{Version: CurrentVersion, Tasks: []Task{
		todo("t-late", 200),
		done("d-new", t2),
		todo("t-early", 100),
		done("d-old", t1),
	}}
	Normalize(f)
	got := ids(f)
	want := []string{"d-old", "d-new", "t-early", "t-late"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("normalized order = %v, want %v", got, want)
		}
	}
	// Idempotent.
	Normalize(f)
	if got2 := ids(f); !equal(got2, want) {
		t.Fatalf("re-normalized order = %v, want %v", got2, want)
	}
}

func TestComputeInsertOrderHeadTail(t *testing.T) {
	f := &File{Version: CurrentVersion, Tasks: []Task{
		todo("a", 100), todo("b", 200),
	}}
	if o, _, _ := mustOrder(t, f, "head", ""); o != 100-orderStep {
		t.Errorf("head order = %d, want %d", o, 100-orderStep)
	}
	if o, _, _ := mustOrder(t, f, "tail", ""); o != 200+orderStep {
		t.Errorf("tail order = %d, want %d", o, 200+orderStep)
	}

	// Empty list: head and tail both seed at orderStep.
	empty := &File{Version: CurrentVersion}
	if o, _, _ := mustOrder(t, empty, "head", ""); o != orderStep {
		t.Errorf("empty head order = %d, want %d", o, orderStep)
	}
}

func TestComputeInsertOrderAfterTodoMidpoint(t *testing.T) {
	f := &File{Version: CurrentVersion, Tasks: []Task{
		todo("a", 0), todo("b", orderStep),
	}}
	o, fallback, err := ComputeInsertOrder(f, "after", "a")
	if err != nil || fallback {
		t.Fatalf("unexpected err=%v fallback=%v", err, fallback)
	}
	if o != orderStep/2 {
		t.Errorf("after order = %d, want %d", o, orderStep/2)
	}
}

func TestComputeInsertOrderAfterLastTodo(t *testing.T) {
	f := &File{Version: CurrentVersion, Tasks: []Task{todo("a", 100)}}
	o, _, err := ComputeInsertOrder(f, "after", "a")
	if err != nil {
		t.Fatal(err)
	}
	if o != 100+orderStep {
		t.Errorf("after-last order = %d, want %d", o, 100+orderStep)
	}
}

func TestComputeInsertOrderRenumbersOnExhaustedGap(t *testing.T) {
	// Adjacent todos with no integer between them force a renumber.
	f := &File{Version: CurrentVersion, Tasks: []Task{
		todo("a", 5), todo("b", 6), todo("c", 7),
	}}
	o, fallback, err := ComputeInsertOrder(f, "after", "a")
	if err != nil || fallback {
		t.Fatalf("unexpected err=%v fallback=%v", err, fallback)
	}
	// After renumber, a=orderStep, b=2*orderStep, midpoint between them.
	if f.Tasks[0].Order != orderStep {
		t.Errorf("renumbered a order = %d, want %d", f.Tasks[0].Order, orderStep)
	}
	if o <= orderStep || o >= 2*orderStep {
		t.Errorf("after order = %d, want strictly between %d and %d", o, orderStep, 2*orderStep)
	}
}

func TestComputeInsertOrderAfterDoneFallsBackToHead(t *testing.T) {
	f := &File{Version: CurrentVersion, Tasks: []Task{
		done("d", time.Now()), todo("a", 100),
	}}
	o, fallback, err := ComputeInsertOrder(f, "after", "d")
	if err != nil {
		t.Fatal(err)
	}
	if !fallback {
		t.Error("expected fallbackHead = true for done target")
	}
	if o != 100-orderStep {
		t.Errorf("fallback head order = %d, want %d", o, 100-orderStep)
	}
}

func TestComputeInsertOrderAfterMissing(t *testing.T) {
	f := &File{Version: CurrentVersion, Tasks: []Task{todo("a", 100)}}
	_, _, err := ComputeInsertOrder(f, "after", "nope")
	if !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("err = %v, want ErrTaskNotFound", err)
	}
}

func mustOrder(t *testing.T, f *File, pos, after string) (int, bool, error) {
	t.Helper()
	o, fb, err := ComputeInsertOrder(f, pos, after)
	if err != nil {
		t.Fatalf("ComputeInsertOrder(%q,%q): %v", pos, after, err)
	}
	return o, fb, err
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
