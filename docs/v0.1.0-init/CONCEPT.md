Concept - the initial concept for v0.1.0
---
 Microbeads
• Minimal beads-inspired task tracker
• microbeads is not for tracking your messy feature ideas. It's for encoding 
• Two statuses: todo/done.
• commands: 
  • mb add head/tail/after [id]
  • mb get head/id
  • mb list
  • mb done
  • mb delete [id]
  • mb prune
  • mb init
  • mb onboarding
• mb init is to add instructions to AGENTS.md etc, within <mb-instructions> tag. mb onboarding gives agent instructions for using mb. 
• no box drawing, no JSON output, just markdown
• store in JSON, no dolt, no sqlite, no persistent process
• only specify title and description on creation. Description can have line breaks. Accepts either --title --description flags or --json.
• id: first four chars of folder name, dash, four char hash of (title, created at, first 200 chars of description). Enough to avoid collision, no duplication with title, 
• data:
  • id
  • title
  • description 
  • isDone
  • created at
  • updated at
  • completed at
• store in .beads/mb.json from repo root. If no .beads folder at current dir, traverse up to nearest parent with .beads, stopping if folder with .git found and creating if necessary 
• pass -f to reference specific file - e.g. backend and frontend tasks in monorepo. If only filename given, will use that file from regular .beads path. .JSON extension added automatically 
• no blocked status. If there's a blocker add instructions to head, whether to fix or wait
• no in progress or pending status. Microbeads is single-threaded; the current agent should always be working on the head task. 
• do I do init / onboarding, or on/off to add/remove from agents.md?
• mb init string idea:
  • this project uses microbeads, a minimal task tracker. If the user references "microbeads", "beads", "mb", "check for next task", or similar, run `mb onboarding` to get started.
• mb onboarding string idea:
  • briefly document commands
  • if you have completed a bead, you MUST run `mb done`
  • when working on beads, commit after each completed task unless the user has specified otherwise in prompt or agents.md
  • if you encounter a blocker that prevents you from completing your current task this session, add a new task to head so next session will address it
• support hooks, defined in .beads/mb-hooks.json - shell commands to trigger before or after running mb commands, optionally passing output in after the mb commands, use $id etc to access fields to pass into the hooks
  • E.g. hook on mb done to git add and git commit using id of completed task; hook on mb onboarding to cat your custom mb instructions and run after output; hook on mb worktree (unused command by mb) which creates new worktree, cd's into it, runs pnpm install
  • hooks have title and description 
  • rally uses hooks which trigger "rally robots ..." commands - agents don't call rally directly
  • how to distribute alongside rally?
