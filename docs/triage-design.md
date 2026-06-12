# Triage State Machine — Design (B-03)

**Status:** Sign-off artifact for B-04. Owner: WS-B (Vox/Dialogist).
**Scope:** Schema, symptom fork, agent contract, mapping of every `troubleshooting.md` §1 branch + pre-flight + escalation terminal onto the schema, and the SessionTable/Lex-attribute mirroring plan.
**Code:** `internal/triage/types.go` (types only; engine + tree data are B-04 in `engine.go`/`trees.go`/`classify.go`).

---

## 1. Design stance

The eight diagnostic trees in `troubleshooting.md` §1, the Universal Pre-Flight Checklist, and the "When to Escalate" criteria are **data, not prose to improvise over**. They are encoded as a deterministic state machine in `internal/triage`.

- **The triage engine OWNS navigation** — which tree, which step, which transition fires on a given outcome, which terminal is reached. It is the single source of truth for "where are we in troubleshooting."
- **The Bedrock agent ONLY renders/explains** the current step's content in persona voice and answers free-form questions grounded in the KB. It never decides the next step.

Session truth lives in `SessionTable`, keyed by the Lex `sessionId` (= Connect `ContactId`), and is mirrored into Lex session attributes each turn so the stateless Lambda can resume mid-tree.

---

## 2. Schema (trees-as-data)

All types live in `internal/triage/types.go`.

| Type | Role | Key fields |
|---|---|---|
| `SymptomFork` | Top-level routing table | `PreFlight *Tree`, `Trees map[SymptomClass]*Tree` |
| `SymptomClass` (string enum) | The 8 quick-index symptoms + `preflight` | one const per tree; `AllSymptomClasses` (8, no catch-all) |
| `Tree` | One diagnostic flow | `ID`, `Symptom`, `Title`, `Goal`, `EntryStepID`, `Steps map[StepID]*Step` |
| `Step` | A decision point read aloud | `ID`, `Ordinal`, `ReadAloudKey`, `KBDocRef`, `OnWorked`, `OnDidntWork`, `OnUnclear` |
| `StepID` (string) | Stable per-tree id | e.g. `tree1.s2`; stored in `attempted_steps` and logs |
| `Outcome` (string enum) | User verdict the agent returns | `worked` / `didnt_work` / `unclear` |
| `Transition` | Deterministic edge for one outcome | exactly one of `NextStep *StepRef` or `Terminal *Terminal` |
| `StepRef` | Pointer to a step (maybe cross-tree) | `TreeID` (empty = this tree), `StepID` |
| `Terminal` | A leaf | `Kind`, `Reason`, `Priority`, `Disposition`, `RouteTo`, `ReadAloudKey` |
| `TerminalKind` (enum) | How a flow ends | `resolved` / `escalate` / `rma` / `route_to_tree` |
| `EscalationReason` (enum) | Why it escalates | 3 shared with `escalation.go` + 7 tree-driven; no catch-all |
| `PromptRef` (string) | Key into the read-aloud prompt catalog | verbatim §1 branch text rendered in persona voice |
| `KBDocRef` (string) | KB doc for grounding free-form answers | e.g. `windows/2.4-mic-privacy` |

### Schema invariants (validated at engine init in B-04)
1. Every `SymptomClass` in `AllSymptomClasses` has an entry in `SymptomFork.Trees`.
2. Every `Step` defines all three outcome transitions (`OnWorked`, `OnDidntWork`, `OnUnclear`) — **no implicit fall-through, no catch-all**.
3. Every `Transition` sets exactly one of `NextStep` / `Terminal`.
4. Every `NextStep`/`RouteTo` `StepRef` resolves to a real `Tree.Steps` entry.
5. `Terminal.Kind` constrains which fields are meaningful (Resolved ⇒ `Disposition`; Escalate/RMA ⇒ `Reason`+`Priority`; RouteToTree ⇒ `RouteTo`).

### Why references, not inline text
`ReadAloudKey` and `KBDocRef` are **keys**, not strings of copy. This keeps the data graph small, lets WS-E swap pacing/phrasing without touching the topology, keeps the source-of-truth branch copy in one catalog, and lets the agent fetch persona-rendered phrasing and ground answers against the A-01 KB splits.

---

## 3. The symptom fork

Two phases:

1. **Pre-flight always runs first.** `SymptomFork.PreFlight` is a `Tree` (`ID: "preflight"`, `Symptom: SymptomPreFlight`) encoding the 6-item Universal Pre-Flight Checklist. Its terminals either **resolve** the ticket (`TerminalKindResolved`) or **route into a symptom tree** (`TerminalKindRouteToTree`, `RouteTo` → the selected tree's `EntryStepID`).
2. **Symptom routing.** When pre-flight does not resolve, the B-04 classifier (slot/keyword first, single Haiku `Converse` fallback per B-04) produces a `SymptomClass`; the engine selects `SymptomFork.Trees[class]`. The eight index utterances each map to exactly one class — there is **no "misc" class**.

| # | User says | `SymptomClass` | Tree |
|---|---|---|---|
| 1 | "No sound in my headset" | `no_audio_output` | Tree 1 |
| 2 | "They can't hear me" | `mic_not_working` | Tree 2 |
| 3 | "Not showing up / not detected" | `not_detected` | Tree 3 |
| 4 | "Only one ear / mono" | `one_sided_audio` | Tree 4 |
| 5 | "Choppy/robotic/crackly/echo" | `distorted_audio` | Tree 5 |
| 6 | "Too quiet/loud / hear myself" | `volume_sidetone` | Tree 6 |
| 7 | "Mute out of sync / buttons dead" | `mute_call_control` | Tree 7 |
| 8 | "Keeps cutting out / dropping" | `intermittent_drops` | Tree 8 |

---

## 4. The agent contract (who says what)

Each turn the engine and the Bedrock agent exchange exactly two payloads.

**Engine → agent: `StepView`** (read-only description of the current step)
- `TreeID`, `TreeTitle`, `Goal`, `StepID`, `Ordinal`, `TotalSteps` — orientation ("step N of M").
- `ReadAloud PromptRef` — the prompt-catalog key the agent renders in persona voice.
- `KBDoc KBDocRef` — the doc to ground free-form follow-ups against (may be empty).
- `IsTerminal` + `Terminal` — when a leaf has fired, the agent reads the terminal's `ReadAloudKey` (the resolved/handoff line) and the contact closes/transfers.

The agent **renders and explains** this. It does **not** choose the next step.

**Agent → engine: `TurnResult`**
- `Outcome` — `worked` / `didnt_work` / `unclear`. **The only field that drives navigation.**
- `FreeFormHandled` — true if the agent answered a side question this turn (grounded in `KBDoc`) without a verdict yet; the engine re-presents the same step rather than advancing.
- `RawUtterance` — transcript retained for telemetry and the escalation/frustration detector; not used for navigation.

**Navigation is deterministic:** the engine maps `Outcome` → `Step.OnWorked|OnDidntWork|OnUnclear` → `Transition` → (`NextStep` or `Terminal`). `unclear` re-prompts once (`UnclearRepromptLimit`/`UnclearStreak`), then the next unclear is treated as `didnt_work` (B-05 behavior).

This realizes the WS-A division (A-08): RAG is the explainer, the triage engine is the navigator.

---

## 5. Mapping every §1 branch onto the schema

Each tree below is a `Tree`; each numbered item is a `Step`; each **If YES / If NO** is a `Transition`; each "→ resolved / → Tree N / → When to Escalate / → RMA" is the corresponding `Terminal`/`NextStep`. **Every leaf maps — there is no catch-all.** `R` = `OutcomeWorked`, `N` = `OutcomeDidntWork`. Terminal kinds: `Resolved`, `Escalate(reason)`, `RMA(reason)`, `RouteToTree`.

### Pre-Flight (`preflight`)
6 checklist items as ordered steps (reseat, default device, hardware mute, volume, single-app-mic, reboot). Each item: `R → Resolved(contained_resolved)`. The terminal step: `N → RouteToTree(RouteTo = classifier-selected tree entry)`. (The classifier runs at the route point; pre-flight never dead-ends.)

### Tree 1 — No Audio Output (`tree1`)
- s1 recognized? `N → RouteToTree(tree3)`; `R → s2`
- s2 headset = Windows output? `R → Resolved`; `N → s3`
- s3 volume/mute check? `R → Resolved`; `N → s4`
- s4 playback test outside softphone? `R(hear test) → s5`; `N(silent) → s6`
- s5 softphone output selection (§4)? `R → Resolved`; `N → Escalate(genesys_platform, medium)`
- s6 Windows-layer remediation (troubleshooter→enable→port→reboot→driver, §2)? `R → Resolved`; `N → Escalate(tree_exhausted, medium)` (after reboot+driver, suspect hardware → see reboot-limit rule §6)

### Tree 2 — Mic / Other Party Can't Hear (`tree2`)
- s1 hardware mute? `R → Resolved`; `N → s2`
- s2 headset = Windows input? `R → Resolved`; `N → s3`
- s3 input level meter moves? `R(moves) → s4`; `N(no move) → s5`
- s4 softphone mic + browser permission + in-call mute (§4)? `R → Resolved`; `N → Escalate(genesys_platform, medium)`
- s5 Windows mic privacy + close apps + port + reboot + driver (§2)? `R → Resolved`; `N → RMA(hardware_fault, high)` (level meter never moves after reboot+driver+known-good port → failed mic/boom)

### Tree 3 — Not Detected (`tree3`)
- s1 reseat direct? `R(appears) → RouteToTree(original symptom tree1/tree2)`; `N → s2`
- s2 different USB port? `R → Resolved`; `N → s3`
- s3 different computer/cable? `R(works elsewhere) → s4`; `N(fails everywhere) → RMA(hardware_fault, high)`
- s4 Device Manager (yellow !/absent)? `R(fixed via uninstall/scan/driver) → Resolved`; `N → s5`
- s5 managed-machine/policy? `R(policy suspected) → Escalate(managed_machine_policy, medium)`; `N(known-good, no policy) → RMA(hardware_fault, high)`

### Tree 4 — One-Sided / Mono (`tree4`)
- s1 Windows balance L/R equal? `R → Resolved`; `N → s2`
- s2 Mono-audio toggle off? `R → Resolved`; `N → s3`
- s3 stereo test outside softphone? `R(both ears) → Resolved` (call audio is mono — expected; reassure); `N(still one-sided) → s4`
- s4 wiggle test / cable / port? `R(cuts with movement → cable) → RMA(hardware_fault, high)`; `N(consistently dead one ear → failed driver) → RMA(hardware_fault, high)`

### Tree 5 — Distorted / Choppy / Echo (`tree5`)
- s1 which best describes it? (3-way branch via the prompt) — `robotic/choppy → s2`, `crackling/static → s3`, `echo → s4`. Encoded as a router step whose three transitions are `NextStep`s (the user's pick is the "outcome"); see §7 on multi-way branches.
- s2 network/CPU (wired, off VPN, free CPU, §4 media test)? `R → Resolved`; `N(persists on good wired) → Escalate(genesys_platform, medium)`
- s3 USB/driver (reseat, port, disable enhancements, lower sample rate, driver, §2)? `R → Resolved`; `N(persists multi-port + driver) → RMA(hardware_fault, high)`
- s4 echo — who hears it? user-hears-self → `NextStep(tree6 sidetone, s3)`; other-party → s4b
- s4b headset-not-speakers + lower volume + AEC on (§4)? `R → Resolved`; `N(persists with proper headset) → Escalate(genesys_platform, medium)`

### Tree 6 — Volume / Sidetone (`tree6`)
- s1 other-party volume, or hearing self? `other → s2`, `self/sidetone → s3` (router step)
- s2 call volume (dial, Windows mixer, levels, loudness EQ, §2/§4)? `R → Resolved`; `N(max still too quiet all apps) → Escalate(tree_exhausted, medium)` (after driver update)
- s3 sidetone (vendor app lower/raise; Windows Listen-tab uncheck, §3/§2)? `R → Resolved`; `N(feature present but control no effect) → Escalate(genesys_platform, medium)` (vendor firmware/driver, §3). For models with no sidetone feature: prompt explains it's expected → `Resolved` (expectation set).

### Tree 7 — Mute Sync & Call Control (`tree7`)
- s1 buttons dead, or mute out of sync? `buttons → s2`, `desync → s5` (router step)
- s2 model on supported call-control list (§4/§3)? `R(supported) → s3`; `N(not supported) → Resolved` (expected; user uses in-app controls — expectation set, audio works)
- s3 vendor app installed/running (§3)? `R → Resolved`; `N → s4`
- s4 restart link (re-plug, restart vendor app, restart Genesys, §4)? `R → Resolved`; `N → Escalate(genesys_platform, medium)`
- s5 re-sync by toggling each side + fresh test call? `R(track together) → Resolved`; `N(keep drifting) → s6`
- s6 vendor app present + supported model + restart link? `R → Resolved`; `N → Escalate(genesys_platform, medium)` (firmware/middleware; safety call-out: trust in-app mute)

### Tree 8 — Intermittent Disconnects (`tree8`)
- s1 off the dock/hub, direct port? `R(drops stop) → Resolved`; `N → s2`
- s2 different direct port? `R → Resolved`; `N → s3`
- s3 disable USB power mgmt / selective suspend + USB/chipset/dock firmware (§2)? `R → Resolved`; `N → s4`
- s4 wiggle test / different cable / different computer? `R(movement triggers / drops on another machine → hardware) → RMA(hardware_fault, high)`; `N(only this machine after all above) → Escalate(managed_machine_policy_or_host, medium)` mapped to `ReasonGenesysPlatform`→ actually host/USB-controller; uses `Escalate(tree_exhausted, medium)` with handoff note "host USB controller / dock firmware / driver beyond the desk".

> Tree-8 s4 "only this machine" terminal: escalates to Tier 2 / IT (host controller / dock firmware / driver), reason `tree_exhausted`, priority medium. This is a Tier-2 escalation, not RMA, because hardware tested good elsewhere.

---

## 6. Escalation terminals — the "When to Escalate" criteria

The §1 "When to Escalate" list maps to `EscalationReason` consts (no catch-all). The first three are **shared verbatim** with `internal/handlers/escalation.go` so the engine and the keyword/frustration detector share one vocabulary:

| §1 criterion | `EscalationReason` | `TerminalKind` | Priority |
|---|---|---|---|
| (user types "agent/human") | `user_requested` | escalate | high |
| (≥3 frustration phrases) | `user_frustrated` | escalate | medium |
| (≥5 failed steps) | `troubleshooting_exhausted` | escalate | medium |
| Hardware fault suspected (ports/2nd machine/wiggle/dead earcup/detection fails everywhere) | `hardware_fault` | **rma** | high |
| **"≥ 2 reboots / no progress" rule** | `reboot_limit` | escalate | medium |
| Verified-failed fix after completing a tree (incl. reboot + driver reinstall) | `tree_exhausted` | escalate | medium |
| Managed/locked-down machine, no admin, USB whitelisting/policy | `managed_machine_policy` | escalate | medium |
| User can't/won't perform the steps | `user_cannot_perform` | escalate | medium |
| Accessibility need / accommodation | `accessibility_need` | escalate | medium |
| Suspected defective unit / out-of-box / physical damage / fails on known-good machine | `hardware_fault` | **rma** | high |
| Beyond-the-desk Genesys/platform (good wired, headset confirmed good) | `genesys_platform` | escalate | medium |

### The "≥ 2 reboots / no progress" rule (mechanized)
`TriageState` carries `RebootCount` and `DriverReinstalled`. Steps that instruct a reboot/driver reinstall increment/set these via the engine (B-04). The const `RebootLimit = 2`: when `RebootCount >= RebootLimit && DriverReinstalled` and the current step would otherwise loop back to the same remediation, the engine fires `Escalate(reboot_limit, medium)` **instead of** re-running the steps. This is a cross-cutting guard the engine applies before any `OnDidntWork` transition into a reboot/driver step, so it does not need to be re-encoded on every leaf.

### RMA terminals
`TerminalKindRMA` is kept distinct from `TerminalKindEscalate` so the warm-transfer/handoff path (OPS-2) can branch on "open a defective-unit / RMA case" vs "warm-transfer to Tier 2." Both carry `ReasonHardwareFault`. RMA leaves: Tree 2 s5, Tree 3 s3/s5, Tree 4 s4, Tree 5 s3, Tree 8 s4.

### Clean handoff
Every escalation/RMA terminal's `ReadAloudKey` points at a handoff line, and the engine emits the §1-mandated handoff context from `TriageState`: symptom, **exact tree + step stopped at** (`CurrentTree`/`CurrentStep`), `AttemptedSteps`, `RebootCount`/`DriverReinstalled`, headset make/model (from slots, B-07), and managed-machine flag. This populates the OPS-2 warm-transfer context so Tier 2 doesn't restart at step 1.

---

## 7. Multi-way branch steps (Trees 5 & 6)

Tree 5 s1 (robotic / crackling / echo) and Tree 6 s1 (other-party volume / sidetone) and Tree 7 s1 (buttons / desync) and Tree 5 s4 (who hears echo) are **classification forks**, not yes/no. They are modeled as ordinary `Step`s whose `ReadAloud` asks the user to characterize the symptom; the agent returns the user's pick, which the engine maps onto the step's transitions. For these forks the engine reads the pick from `TurnResult.RawUtterance` via the same B-04 classifier (a sub-classification), then takes the corresponding `NextStep`. The `OnWorked`/`OnDidntWork`/`OnUnclear` triplet is still populated (`OnUnclear` re-asks the characterization; `OnWorked` is unused/aliased to the most-common branch only if the user pre-answers a fix). This keeps the schema uniform (three transitions per step) while supporting the handful of n-way forks. **B-04 note:** if a cleaner representation is desired, add an optional `Branches map[string]*StepRef` to `Step` in B-04; B-03 intentionally keeps `Step` minimal and routes forks through the classifier to avoid a second navigation mechanism.

---

## 8. SessionTable / Lex-attribute mirroring plan

**Source of truth:** `SessionTable` (DynamoDB), keyed by Lex `sessionId` (= Connect `ContactId`). The B-01 session store does `Load`/`Save` of `models.Session`; `TriageState` is the navigation payload.

**`TriageState` fields** (struct tags in `types.go` match the B-01 typed-accessor list): `current_tree`, `current_step`, `symptom`, `attempted_steps`, `failed_steps`, `frustration_count`, `reboot_count`, `driver_reinstalled`, `unclear_streak`, `last_response`, `resolved`, `escalated`, `escalation_reason`. (B-01 also owns `pace_rate`, `low_asr_count`, `no_match_count` on the broader session; those are VX-5/VX-6 concerns and live alongside, not inside, `TriageState`.)

**Mirroring (each turn):**
1. **Load** `TriageState` from `SessionTable` at the top of the Lex handler (B-02). On read failure, degrade gracefully and log (B-01 DoD).
2. **Project** the hot subset into Lex session attributes via the `Attr*` key consts (`current_tree`, `current_step`, `symptom`, `failed_steps`, `frustration_count`, `reboot_count`, `escalation_reason`, `resolved`, `escalated`). This lets the stateless Lambda resume mid-tree without a second read and feeds the existing `lex-lambda/main.go` reads of `frustration_count`/`failed_steps` (fixing PRD honest-truth #4: those attributes are now actually written).
3. **Persist** the full `TriageState` back to `SessionTable` after producing the response — including on escalation/Close — with a TTL refresh and a conditional write on `LastActivity` for concurrent-turn safety (B-01).
4. **Escalation handoff:** when an Escalate/RMA terminal fires, the engine sets `escalated`/`escalation_reason` in state and the handler writes the OPS-2 warm-transfer attributes (`escalation_requested`, `escalation_reason`, `escalation_priority`) exactly as `BuildEscalationResponse` already does — now driven by a real reason from the tree rather than a discarded count.

**Why mirror instead of attributes-only:** Lex session attributes are size-limited and string-typed; `AttemptedSteps` (a growing list) and the full structured state belong in DynamoDB. Attributes carry only the few scalars the hot path and Connect flow need.

---

## 9. Sign-off checklist for B-04

- [x] Trees-as-data schema: `Tree`/`Step`/`Transition`/`Terminal`/`StepRef` with stable `StepID`, `ReadAloudKey` (`PromptRef`), `KBDocRef`, and yes/no/other transitions.
- [x] Top-level symptom fork mapping the 8 quick-index symptoms + always-first pre-flight (`SymptomFork`, `SymptomClass`, `AllSymptomClasses` — no "misc").
- [x] Agent contract: `StepView` (engine→agent) and `TurnResult` (agent→engine, `worked`/`didnt_work`/`unclear`); engine owns navigation, agent renders/explains.
- [x] Escalation terminals incl. the "≥ 2 reboots / no progress" rule (`RebootLimit`, `RebootCount`, `DriverReinstalled`, `ReasonRebootLimit`) and RMA terminals (`TerminalKindRMA`, `ReasonHardwareFault`); reasons aligned with `escalation.go`.
- [x] Every §1 branch (pre-flight + Trees 1–8 + every leaf) mapped to a schema instance; no catch-all.
- [x] SessionTable/Lex-attribute mirroring plan, keyed by Lex `sessionId`.
- [x] `go build ./internal/triage/ && go vet ./internal/triage/` pass clean.
