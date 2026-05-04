# Thematic Universe Brainstorm

## The Anchor: Rally's Running Primitives

Rally's tries/runs/relays/runners are firmly athletics. This is the strongest existing thread, so each theme below is evaluated by how naturally it extends from that anchor.

---

## Theme A: Track & Field 🏟️

The most natural extension. Rally already lives here — just lean in fully.

| Tool | Current | Proposed | Metaphor |
|------|---------|----------|----------|
| Sandbox manager | dune | **track** | The surface/venue — you lay down a track before anyone runs |
| Orchestrator | rally | rally (keep) | The athletic meet/event |
| Task tracker | laps | **laps** (`lp`) | Circuits completed around the track |

**Rally primitives — no changes needed:**
- runners → athletes doing the work
- tries → individual attempts
- runs → race executions  
- relays → team handoffs between runners

**Expansion vocabulary (all single-domain, all intuitive):**

| Term | Potential use | Notes |
|------|--------------|-------|
| **heats** | Batch/grouped runs | Already noted by you — rounds of competition |
| **lanes** | Parallel execution streams | Each agent gets a lane; natural isolation metaphor |
| **splits** | Timing/metrics/progress | "Split times" = measuring intermediate performance |
| **pace** | Velocity tracking | "Setting the pace" |
| **hurdles** | Blockers/obstacles | Visual, immediately understood |
| **strides** | Larger work units / epics | Bigger than a lap |
| **blocks** | Starting blocks → initialization / setup | `track blocks` for env scaffolding? |
| **batons** | Context passed between runners | Data/state handoff in relays |
| **field** | The broader project workspace | "Field events" = non-track work |

**Strengths:**
- Rally's existing primitives are *already* track & field — zero friction
- Every expansion term is instantly understood; no explanation needed
- "Track" for sandbox manager is solid: you set up a track, then race on it
- `track init`, `track create` reads naturally as environment setup
- **laps** is the perfect task unit: sequential, bounded, small, countable

**Concerns:**
- "Track" as a tool name could be confused with `git` tracking or issue trackers
- The athletics metaphor might feel too "sporty" for some users
- `track` is 5 letters / 1 syllable — fine, but not as punchy as `dune`

**Sentence test:** *"Spin up a track, start a rally, knock out some laps."* ✅ Flows perfectly.

---

## Theme B: Endurance Motorsport 🏎️

Rally IS a motorsport term. Lean into that — specifically endurance racing (Le Mans, Dakar) where driver rotation maps 1:1 to agent handoffs.

| Tool | Current | Proposed | Metaphor |
|------|---------|----------|----------|
| Sandbox manager | dune | **garage** (`gr`) | Where you build/maintain the vehicle (environment) |
| Orchestrator | rally | rally (keep) | The race event — rally is literally motorsport |
| Task tracker | laps | **laps** (`lp`) | Circuits of the course |

**Bonus:** The **Dakar Rally** is literally a rally through sand dunes. So `dune` already accidentally fits this theme if you squint — dune is the terrain, rally is the race through it.

**Rally primitives — reframed but compatible:**
- runners → drivers
- tries → qualifying attempts
- runs → race stints
- relays → driver handoffs (in endurance racing, drivers literally relay!)

**Expansion vocabulary:**

| Term | Potential use | Notes |
|------|--------------|-------|
| **stints** | Work sessions between pit stops | A defined period behind the wheel |
| **pit** | Quick env fixes / hot-swap | Pit stop = fast environment adjustment |
| **grid** | Starting positions / task queue ordering | "On the grid" |
| **draft** | Building on previous agent's work | Slipstreaming = using prior momentum |
| **flags** | Status signals (green/yellow/red) | Race flags → task/system status |
| **sectors** | Code areas / workspace regions | Track divided into sectors |
| **chicane** | Deliberate slowdowns / review gates | Forced caution points |

**Strengths:**
- Rally literally means motorsport rally — no metaphor stretching at all
- Endurance racing's driver relay model is a *perfect* 1:1 map for AI agent handoffs
- **garage** for sandbox management is very intuitive: "spin up a garage," "tear down the garage"
- Dakar Rally connection means `dune` retroactively fits (accidental coherence!)
- Rich, deep vocabulary for expansion

**Concerns:**
- Requires reframing tries/runs/relays slightly (from athletics to motorsport)
- "Garage" is 2 syllables
- Motorsport may feel less accessible than athletics to non-racing fans
- Some terms (grid, flags) have existing tech meanings

**Sentence test:** *"Set up a garage, fire up the rally, finish your laps."* ✅ Works well.

---

## Theme C: Trail Running / Orienteering 🏔️

Rally as a "rally point" — a place where the group gathers before heading out. Each task is a marker on the trail.

| Tool | Current | Proposed | Metaphor |
|------|---------|----------|----------|
| Sandbox manager | dune | **camp** | Base camp — where you prepare and stage |
| Orchestrator | rally | rally (keep) | Rally point — gathering and coordination |
| Task tracker | laps | **blazes** (`bz`) or **cairns** (`cn`) | Trail markers left to guide the path |

**Blazes** are paint marks on trees that mark a trail. **Cairns** are stacked-stone markers. Both are small, sequential, and guide progress through wilderness.

**Rally primitives — reframed:**
- runners → trail runners / scouts
- tries → route attempts
- runs → trail traversals
- relays → handoffs between scout teams

**Expansion vocabulary:**

| Term | Potential use | Notes |
|------|--------------|-------|
| **waypoints** | Milestones / checkpoints | GPS waypoints along the route |
| **ridgeline** | Overview / dashboard | High-level view of progress |
| **switchbacks** | Direction changes / pivots | When the path reverses |
| **summit** | Project completion / goal | The destination |
| **bearing** | Direction / priority | "Set a bearing" |
| **traverse** | Cross-cutting work | Moving across the face of a challenge |

**Strengths:**
- "Camp" for sandbox management is lovely: "set up camp," "break camp"
- "Rally point" is military/outdoor — strong coordination metaphor
- Blazes/cairns are beautiful, distinct concepts with no tech clashes
- The exploration framing suits open-ended AI coding work (you're finding the path)

**Concerns:**
- `bz` and `cn` are less natural than `lp` or `tk` as CLI shorthands
- "Blazes" is close to "blazingly fast" Rust marketing; "cairns" is slightly obscure
- Trail metaphor is weaker for the sequential queue (trails branch; your task queue doesn't)
- Biggest stretch of the three themes — rally's running primitives map less cleanly

**Sentence test:** *"Set up camp, rally the runners, follow the blazes."* ⚠️ Works but feels a bit forced.

---

## Theme D: Don't Force It 🎯

Intentionally heterogeneous. Each tool gets its best standalone name.

| Tool | Name | Why |
|------|------|-----|
| Sandbox manager | **dune** (keep) | Already great. Sand → sandbox, bigger + more built-in |
| Orchestrator | **rally** (keep) | Strong. Mixed metaphor is a feature, not a bug |
| Task tracker | **tacks** (`tk`) | Best standalone name: sharp, purposeful, zero clashes, great CLI |

**The argument for this:**
- Forced theming can produce awkward names (picking "garage" because it fits a theme even though "dune" is better as a name)
- Users learn tool names individually, not as a thematic set
- `dune`, `rally`, `tacks` are all strong *individual* names
- You can still use rally-adjacent vocabulary (heats, lanes) without the task tracker being theme-locked

**The argument against:**
- Loses the "aha" moment of a unified metaphor
- Harder to name future tools — no theme to draw from
- Documentation and branding feel less cohesive

---

## Comparison Matrix

| | Track & Field | Motorsport | Trail | Don't Force It |
|--|---------------|------------|-------|----------------|
| **Task tracker name** | laps (`lp`) | laps (`lp`) | blazes (`bz`) | tacks (`tk`) |
| **Sandbox name** | track | garage | camp | dune (keep) |
| **Rally primitive fit** | ✅✅ native | ✅ natural reframe | ⚠️ stretch | ✅ N/A |
| **Expansion depth** | ✅✅ rich | ✅✅ rich | ✅ moderate | ⚠️ ad hoc |
| **Naming quality** | ✅ good | ✅ good | ⚠️ mixed | ✅✅ best individual |
| **Cohesion** | ✅✅ seamless | ✅ strong | ✅ moderate | ❌ none |
| **Accessibility** | ✅✅ universal | ✅ good | ✅ good | ✅✅ universal |
| **Risk of forced names** | low | moderate | moderate | none |

---

## My Take

**Track & Field is the clear winner for a unified theme.** Here's why:

1. **Zero friction** — rally's primitives are already athletics. No reframing needed.
2. **"Laps" is the strongest task tracker name regardless** — it just happens to also be the most thematically coherent.
3. **"Track" for sandbox management** is intuitive and one syllable: you set up a track, then race on it.
4. **Expansion is effortless** — heats, lanes, splits, pace, hurdles, strides, blocks, batons are all immediately understood and map to real tool concepts.
5. **The whole sentence writes itself:** *"Lay down a track, start a rally, complete your laps."*

**If unified theming feels too constraining**, Theme D with `tacks` gives you the best individual name. You sacrifice cohesion but gain naming freedom.

**The key question is:** Do you value the "aha" of a unified metaphor enough to rename `dune` → `track`? If yes → Track & Field + laps. If no → keep dune + tacks.

> [!TIP]
> You don't have to decide on the sandbox rename now. You could rename laps → **laps** today (strong standalone AND thematically coherent) and decide on dune → track later. Laps works in both scenarios.
