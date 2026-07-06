package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const commandTimeout = 10 * time.Second

type Runner struct {
	Binary  string
	Timeout time.Duration
}

func (r Runner) Load(ctx context.Context) (Snapshot, error) {
	status, err := r.loadStatus(ctx)
	if err != nil {
		return Snapshot{Missing: true}, err
	}
	rootTasks, err := r.loadTasks(ctx, "list", "--root", "--all", "--json-output")
	if err != nil {
		return Snapshot{Missing: true}, err
	}

	stints := make(map[string]*Stint)
	for i := range status.Stints {
		s := &status.Stints[i]
		stint := s.toStint()
		stints[s.Name] = stint
		if !s.Queued || s.Archived {
			continue
		}
		tasks, err := r.loadTasks(ctx, "-f", stintFileArg(s.Name), "list", "--all", "--json-output")
		if err != nil {
			return Snapshot{Missing: true}, err
		}
		stint.Laps = entriesFromTasks(tasks, nil)
	}

	entries := entriesFromTasks(rootTasks, stints)
	return Snapshot{
		State:   status.State,
		Counts:  status.Counts,
		Claim:   status.Claim,
		Gate:    status.Gate,
		Entries: entries,
	}, nil
}

func (r Runner) loadStatus(ctx context.Context) (statusSnapshot, error) {
	var snapshot statusSnapshot
	out, err := r.run(ctx, "status", "--json-output")
	if err != nil {
		return snapshot, err
	}
	if err := json.Unmarshal(out, &snapshot); err != nil {
		return snapshot, fmt.Errorf("decode status: %w", err)
	}
	return snapshot, nil
}

func (r Runner) loadTasks(ctx context.Context, args ...string) ([]task, error) {
	var response listResponse
	out, err := r.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(out, &response); err != nil {
		return nil, fmt.Errorf("decode list: %w", err)
	}
	if response.Tasks == nil {
		response.Tasks = []task{}
	}
	return response.Tasks, nil
}

func (r Runner) Action(ctx context.Context, args ...string) (string, error) {
	out, err := r.runCombined(ctx, args...)
	if err != nil {
		text := strings.TrimSpace(string(out))
		if text == "" {
			text = err.Error()
		}
		return firstLine(text), err
	}
	return firstLine(strings.TrimSpace(string(out))), nil
}

func (r Runner) run(ctx context.Context, args ...string) ([]byte, error) {
	var stderr bytes.Buffer
	out, err := r.command(ctx, args...).OutputWithStderr(&stderr)
	if err != nil {
		text := strings.TrimSpace(stderr.String())
		if text == "" {
			text = err.Error()
		}
		return nil, fmt.Errorf("%s", text)
	}
	return out, nil
}

func (r Runner) runCombined(ctx context.Context, args ...string) ([]byte, error) {
	return r.command(ctx, args...).CombinedOutput()
}

func (r Runner) command(ctx context.Context, args ...string) command {
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = commandTimeout
	}
	binary := r.Binary
	cctx, cancel := context.WithTimeout(ctx, timeout)
	cmd := exec.CommandContext(cctx, binary, args...)
	return command{Cmd: cmd, cancel: cancel}
}

type command struct {
	*exec.Cmd
	cancel context.CancelFunc
}

func (c command) OutputWithStderr(stderr *bytes.Buffer) ([]byte, error) {
	defer c.cancel()
	c.Stderr = stderr
	return c.Output()
}

func (c command) CombinedOutput() ([]byte, error) {
	defer c.cancel()
	return c.Cmd.CombinedOutput()
}

func entriesFromTasks(tasks []task, stints map[string]*Stint) []Entry {
	entries := make([]Entry, 0, len(tasks))
	for i := range tasks {
		t := &tasks[i]
		kind := t.Kind
		if kind == "" {
			kind = kindLap
		}
		entry := Entry{
			Kind:     kind,
			ID:       t.ID,
			Ref:      t.Ref,
			Title:    t.Title,
			Assignee: t.Assignee,
			IsDone:   t.IsDone,
			Order:    t.Order,
		}
		if kind == kindStint {
			name := t.Ref
			if name == "" {
				name = t.Title
			}
			if stints != nil {
				entry.Stint = stints[name]
				if entry.Stint != nil {
					entry.Laps = entry.Stint.Laps
				}
			}
		}
		entries = append(entries, entry)
	}
	return entries
}

func (s *statusStint) toStint() *Stint {
	return &Stint{
		Name:     s.Name,
		Scope:    s.Scope,
		File:     s.File,
		Todo:     s.Todo,
		Done:     s.Done,
		Total:    s.Total,
		Queued:   s.Queued,
		Archived: s.Archived,
		Active:   s.Active,
	}
}

func stintFileArg(name string) string {
	return "stints/" + name + ".laps"
}

func firstLine(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		return strings.TrimSpace(text[:i])
	}
	return text
}
