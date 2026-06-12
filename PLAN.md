# PLAN.md — Headset Support Agent: Implementation Plan

**Repo:** `C:\dev\Headset-Agent` · **Stack:** `agent-headset-prod` (us-east-1) · **Deploys:** GitHub Actions ONLY
**Companion docs:** [`PRD.md`](PRD.md) (requirements), [`troubleshooting.md`](troubleshooting.md) (domain knowledge the bot serves), [`TODO.md`](TODO.md) (original build plan), `CLAUDE.md` (policies).
**Authoring:** Drafted by a Fable persona team (Meridian/delivery, Vox/AI, Bedrock/infra-sec, Echo/CX-accessibility); synthesized here. Implementation tasks are routed across Haiku→Sonnet→Fable→Opus by difficulty.

---

## 0. How to Use This Plan

This file is the **single source of truth** for an autonomous, multi-agent build. No user interaction is expected during execution (see §5 — all blocking questions are surfaced up front). Agents read this file to pick up work and **edit it in place** as they go. Fresh agent? Read §0, find your workstream (§3), find the first `[ ]` task in the lowest open milestone (§2) whose dependencies are `[x]`, set it `[~]`, and start.

### 0.1 Status markers (mandatory, updated in place)

| Marker | Meaning | Rule |
|---|---|---|
| `[ ]` | Todo | Unclaimed. Anyone may claim. |
| `[~]` | In progress | Set the marker AND add a Working Log entry (§4) with timestamp + agent name **before** writing code. |
| `[x]` | Done | Only after verification (test passing, pipeline green, or runtime check) — never on "code written." Log the evidence. |
| `[!]` | Blocked | State the blocker + unblocking task ID inline, e.g. `[!] blocked-by: A-04`. Re-check each session. |

### 0.2 Working-log convention
Every milestone has a **Working Log** subsection (§4). Append-only, newest at bottom. Each entry records: date, agent, task ID, decisions made (and rejected alternatives), files touched, exact commands run, pipeline result (one line), gotchas/landmines, and any task IDs unblocked. **Bar: a fresh agent with zero conversation history must be able to resume from this file alone.** Terse logs are a defect.

### 0.3 Model-routing rubric
Every task names its model. Use the cheapest model that can do the job well; escalate one tier only after a task fails review twice.

| Model | Use for | Examples here |
|---|---|---|
| **Haiku** | Mechanical, low-ambiguity, pattern-following | KB doc chunking/formatting, YAML/JSON resource blocks from a worked example, log-format sweeps, dashboard JSON, doc copy, status updates |
| **Sonnet** | General implementation with a clear spec | Go handlers, DynamoDB session I/O, Connect contact-flow JSON, unit tests, CI workflow edits, alarms |
| **Fable** | Substantial implementation + moderate cross-file reasoning | Triage state machine, streaming-TTS pipeline, warm-transfer plumbing, RAG wiring, SMS fallback, gap-report clustering |
| **Opus** | Hardest reasoning, architecture, security/correctness analysis | Vector-store/cost decision, IAM least-privilege, contact-flow architecture, guardrail design, disposition taxonomy, PII threat model, Path A/B convergence, channel-switch state design, final security review |

### 0.4 Non-negotiable conventions (bake into every task)
1. **No secrets managers.** AWS Secrets Manager is banned (cost). Config/credentials use exactly four mechanisms: **SSM Parameter Store (Standard tier — String/SecureString with the free AWS-managed key)** for runtime config; **Lambda env vars** for non-sensitive settings; **GitHub Actions secrets** for pipeline-side values; **GitHub OIDC** role assumption for AWS auth (no long-lived keys). Any task that would have reached for Secrets Manager states which of the four it used.
2. **GitHub Actions is the only deploy path.** The local AWS CLI is a *different account* — never use it to deploy or to verify deployed state. Verify via pipeline logs + CloudWatch (in the pipeline's account).
3. **arm64 Lambdas**, Go built `GOOS=linux GOARCH=arm64 CGO_ENABLED=0`.
4. **No stubs.** Real implementations only. Blocked? Mark `[!]` and surface the question in §5 — never paper over with a TODO/fallback.
5. **Every push → watch the pipeline → remediate until green** (`/fix-pipeline` available). One-line run status in the Working Log.
6. **Cost-conscious throughout.** This is a personal project. Call out the monthly cost of anything non-trivial and prefer the cheapest correct option. (Recording + Nova Sonic were chosen knowing they raise cost — see §7.)
7. After finishing a task: update marker → write log entry → check whether you just unblocked a `[!]` task and note it.
8. **Run straight through — do not pause for check-ins.** Once execution begins, work through all milestones without stopping to report progress or ask "should I continue?". The *only* valid reasons to pause are: a genuine implementation blocker not covered by this plan's pre-baked decisions/fallbacks, or a destructive/irreversible action needing confirmation. Routine milestone/task completions are **not** pause points. All decisions are baked in (§5) so autonomous execution is possible.
9. **Sub-agent durability** (when fanning work out): track each sub-agent; if no output for 6 min consider it stalled, stop and relaunch with the same prompt; hard-stop at 30 min; up to 2 restarts (3 attempts total) before surfacing the failure; incorporate any partial results into the retry prompt.
10. **Graviton everywhere:** Lambdas build arm64 (§0.4-3); the WS-H Fargate streaming service likewise runs on **arm64/Graviton** unless a dependency forces x86_64.

---

## 1. Workstream Map

Seven parallel workstreams, each run by an agentic team with a lead persona. One owner per requirement ID (cross-cutting work is logged, not co-owned).

| WS | Charter | Owns (PRD IDs) | Lead persona | Default model |
|---|---|---|---|---|
| **WS-A Knowledge & RAG** | Make `knowledge-base/**` + `troubleshooting.md` retrievable and authoritative for the agent. | CC-1, CC-6, CC-7, DM-1…DM-8 | Vox/Librarian | Sonnet (content: Haiku; guardrail/vector-store: Opus) |
| **WS-B Conversation Engine** | Deterministic triage, verified fixes, real memory, working escalation detection, voice UX. | CC-2, CC-3, CC-4, CC-5, VX-1…VX-6 | Vox/Dialogist | Fable |
| **WS-C Contact Center & Connect** | A real human destination: queues, warm transfer, hours, callback, voicemail, CSAT, dispositions. | OPS-1…OPS-10 | Bedrock/Dispatcher | Sonnet |
| **WS-D Security, Privacy & Compliance** | Make prod legal & private; can't take a stubbed path live. | SEC-1…SEC-7 | Bedrock/Warden | Opus (design/review), Sonnet (impl) |
| **WS-E Accessibility & CX Quality** | Callers who can't hear, don't speak English, or aren't technical still get helped; KPIs measurable. | AX-1…AX-6, OPS-6, OPS-7, KPIs | Echo/Advocate | Sonnet |
| **WS-F Observability & Cost** | See every failure and every dollar; emit the PRD §6 KPIs. | REL-2, REL-7 | Bedrock/Lighthouse | Sonnet (alarm boilerplate: Haiku) |
| **WS-G Test & CI/CD + Reliability** | Honest gates: real tests, real coverage, OIDC, no `continue-on-error` lies; Path B safety gate. | REL-1, REL-3, REL-5, SEC-3 | Bedrock/Gatekeeper | Sonnet |
| **WS-H Nova Sonic Path B** | Build the real bidirectional speech-to-speech path (KVS + `InvokeModelWithBidirectionalStream`) and prove it before it serves live calls. | REL-4 (revised), Path B | Vox+Bedrock/Streamwright | Opus (design), Fable (impl) |

### Dependency graph
```
WS-G (honest CI + OIDC) ──► ALL  (no [x] without a real gate; scheduled jobs need the OIDC role)
WS-D (SEC-2 logs, REL-4 fence) ──► WS-B, WS-C, WS-E  (can't ship turns that log PII or route to a stub)
WS-A (CC-1 KB) ──► WS-B (CC-2/CC-3 retrieve against it), WS-E (AX-1 sends KB steps)
WS-C (OPS-1 queue) ──► WS-B escalation handoff (→ OPS-2 context)
WS-B (CC-4 memory) ──► WS-C OPS-2, WS-E E-1.1/E-2.2, WS-F per-turn KPIs
WS-B + WS-C ──► WS-F per-path KPIs, WS-E pace/persona features
```
**Parallel-start set (day 1, no upstream deps):** WS-G-01…06 (tests/CI), WS-D-04/05 (OIDC), WS-A-01/02 (KB content split), B-01/B-03 (session store + triage design), WS-C-05 (single-source the flows), WS-F-01 (SNS).

---

## 2. Milestone Framework

Milestones cut across workstreams and map 1:1 onto PRD §7 phases. A milestone is **open** when the prior one's exit gate is `[x]`. Task details live in §3; here are the gates and contents.

### `[ ]` M0 — Safe & Honest (PRD Phase 0) — *blockers; nothing ships past this without it*
**Definition of Done:** A live call gets a recording/AI consent disclosure first; recordings land encrypted with retention; no raw transcript/caller-ID reaches CloudWatch; an escalation lands in an honestly-closed/voicemail-or-callback Connect queue **with context** (Q-C: bot-only v1); Path B stays behind its safety gate and cannot take a live call until WS-H acceptance passes; CI fails on real test failures; on-call sees Lambda/Bedrock/Connect alarms; the pipeline auths via OIDC with no long-lived keys.

Tasks: WS-D-01/01b/01c (consent + recording storage + Contact Lens), WS-D-02/03 (PII-safe logging), WS-D-04/05 (OIDC), WS-D-10 (no-payment guardrail), WS-C-01…06 (hours→queue→routing→user→single-source flows→fix dead-end transfer), WS-C-07 (warm-transfer context), WS-G-01…06 (tests + honest CI + coverage gate), WS-G-09 (Path B safety gate), WS-F-01/02/04 (SNS + core alarms + placeholder canary).

### `[ ]` M1 — Actually Helpful (PRD Phase 1)
**DoD:** Answers are grounded in the real KB (retrieval hit visible in logs); the bot runs the Tree-1/Tree-2 triage deterministically, confirms "did that fix it?", and remembers attempted steps across turns via `SessionTable`; first audio starts streaming (≤~2s to first audio); callers can barge in; microphone troubleshooting (the #1 complaint) is a first-class flow.

Tasks: A-04/05/06 (vector store + Bedrock KB + ingestion), A-07/08 (collapse to single agent + grounding prompt), A-03 (mic KB docs, DM-1), B-01/02 (session store wired), B-03/04 (triage state machine), B-05 (verify loop), B-06 (real counters, CC-4), B-08 (streaming/partial TTS), B-09 (barge-in), B-14 (engine test suite).

### `[ ]` M2 — Inclusive & Measurable (PRD Phase 2)
**DoD:** A Deaf/HoH caller has a working non-audio path (SMS/chat/email-the-steps); "slow down / repeat" works and persists; every contact gets a disposition + optional CSAT; reprompt ladder + persona IVR live; slots + guardrails active; KB-gap analytics flowing.

Tasks: E-1.* (SMS/chat/email fallback, AX-1), E-2.* (pace commands, AX-2/VX-5), E-7.* (CSAT, OPS-6), E-8.* (disposition taxonomy + emission, OPS-7), E-9.1…9.3/9.5 (golden tests + judge + gap capture), B-07 (slots, CC-5), B-10 (reprompt ladder, VX-3), B-11 (persona IVR, VX-4), B-13 (low-confidence ASR, VX-6), A-09 (guardrails, CC-6), A-10 (retrieval eval gate), WS-F-05/06 (dashboard + X-Ray), WS-F-09 (token caps).

### `[ ]` M3 — Broaden & Optimize (PRD Phase 3)
**DoD:** DECT/Teams-Zoom/firmware/audio-quality domains covered; callbacks, voicemail, repeat-caller, skills routing live; clear-and-slow persona shipped (Spanish deferred to M4 per Q-E); IAM scoped, `/chat`+website locked down, retention/DSAR defined; DLQs/idempotency; cost guardrails + budgets armed.

Tasks: DM-2…DM-5 (WS-A), WS-C-09/10/11/12/13/14 (after-hours/voicemail/callback/CSAT-flow/repeat-caller/skills), E-3.* (clear persona), E-5.* (sensory-cue alternatives), SEC-4/5/6 (WS-D-06/07/08/09), WS-G-07/08 (DLQ + idempotent provisioning), WS-F-07/08/10 (budgets + concurrency + number sweeper), E-9.6/9.7 (weekly gap loop). *(Spanish E-4.* deferred to M4 per Q-E.)*

### `[ ]` M3.5 — Nova Sonic Path B (Q-D: complete now)
**DoD:** Path B serves real bidirectional Nova Sonic audio on a live Connect call; the persona voices work; session/disposition parity with Path A; golden voice scenarios pass; per-path A/B latency report recorded; the `DeployNovaSonicPath` gate flipped to `"true"` with verified rollback-to-Path-A on stream failure.

Tasks: WS-H-01 (design spike — gates everything), WS-H-02…05 (KVS ingest, bidirectional client, playback bridge, persona/session parity), WS-H-06 (acceptance + gate flip). Runs after M1 (shares the grounded agent/KB and session store) and can overlap M2/M3.

### `[ ]` M4 — Later / Could-haves
**DoD:** OPS-10, DM-6/7/8, AX-6, REL-5 groomed with effort + owner, promoted or formally descoped.

Tasks: E-4.* (Spanish + multi-language, deferred from M3 per Q-E), per-path KPI comparison report (WS-F, once WS-H live), E-6.* (TTY/relay), WS-G-11 (DR runbook), remaining Could-haves triage.

---

## 3. Detailed Workstream Tasks

> Format per task: **ID · status · description · files/resources · model · deps · Definition of Done.** Teams expand subtasks in place without changing IDs, models, or the status grammar.

### WS-A — Knowledge & RAG (CC-1, CC-6, CC-7, DM-1…DM-8)

**Architecture decision (CC-7) — collapse to a single tool-using agent.** `create-agents.py` creates Supervisor + 3 sub-agents but never calls `associate_agent_collaborator` — they're dead weight. We will **not** wire collaboration. Rationale: every supervisor→sub-agent hop is a full extra LLM round-trip inside a tight 25s/voice-<3s budget; the three sub-domains share one guide (retrieval already routes between them); escalation is deterministic Go, not an LLM job. The supervisor becomes the only agent with the KB associated; the other three agent definitions are deleted.

**Division with WS-B:** the deterministic triage state machine (WS-B) *navigates*; the Bedrock agent + KB *narrates/explains* the current step and answers free-form questions, grounded in retrieved chunks. RAG is the explainer, not the navigator.

| ID | St | Task | Files / resources | Model | Deps | DoD |
|---|---|---|---|---|---|---|
| A-01 | `[x]` | Split `troubleshooting.md` + existing `knowledge-base/**` into retrieval-sized docs: one per tree (×8), per §2 subsection, per brand (§3), per §4 Genesys topic, + pre-flight + escalation-criteria | new `knowledge-base/{trees,windows,brands,genesys,common}/*.md`; reconcile the 5 existing docs | Haiku split + Sonnet review | — | Each doc ≤~1500 tokens, self-contained, branch text verbatim, diff-audited (no content lost) |
| A-02 | `[x]` | Add `.metadata.json` sidecars (symptom, connection_type, brand, platform, tree_id) for retrieval filtering | `knowledge-base/**/*.metadata.json` | Haiku | A-01 | Fields match WS-B slot value sets (B-07) exactly |
| A-03 | `[ ]` | Author the microphone doc set (DM-1): not-working/too-quiet/muffled/mute-LED/levels-boost/permissions | new `knowledge-base/microphone/*.md` + sidecars | Fable | A-01,A-02 | Six docs in Likely-cause→Check→Steps→Verify format, Win10+Win11 paths, reviewed vs Tree 2 |
| A-04 | `[ ]` | **Decide & provision the vector store** (see §5 Q-A). Spike S3 Vectors for Bedrock KB in us-east-1; else OpenSearch Serverless collection + policies; record decision + monthly cost | `infrastructure/template.yaml` KB section; decision in §5 | Opus | (Q-A) | Vector store deploys green; cost written down; teardown noted in `destroy.yml` |
| A-05 | `[ ]` | `AWS::Bedrock::KnowledgeBase` (Titan Text Embeddings v2) + `DataSource` (KB bucket, ~300-token chunks/20% overlap) + scoped KB role | `infrastructure/template.yaml`; SSM `/kb-id`, `/kb-data-source-id` | Sonnet | A-04 | KB ACTIVE via pipeline; SSM params populated |
| A-06 | `[ ]` | Ingestion automation: s3-sync + `start_ingestion_job` + poll to COMPLETE; wire into deploy.yml (replaces bare `s3 sync`) | new `scripts/sync-knowledge-base.py`; `deploy.yml` | Sonnet | A-05 | KB edits re-ingest automatically; job failure fails the step (no `continue-on-error`) |
| A-07 | `[ ]` | Collapse to single agent: delete the 3 sub-agents, clean orphans, `associate_agent_knowledge_base`, re-prepare + re-alias | `scripts/create-agents.py` | Fable | A-05,A-06 | Exactly 1 prepared agent with KB associated (asserted in-script); SSM agent params still valid so `lex-lambda` needs no change |
| A-08 | `[ ]` | Grounding prompt: answer ONLY from retrieved chunks, name the tree+step, follow If-YES/If-NO, never invent steps, defer navigation to session attrs from WS-B | `create-agents.py` instruction; `internal/agents/bedrock.go` `buildSystemContext()` | Fable | A-07,B-03 | 10-question probe: every answer cites a real tree/step; off-KB → "I don't have that" + escalation, zero fabricated menu paths |
| A-09 | `[ ]` | Bedrock Guardrails (CC-6): denied topics (payments per SEC-7, legal/medical), grounding ≥0.7 / relevance ≥0.5, PII anonymization, profanity; attach to agent | `template.yaml` `AWS::Bedrock::Guardrail`; `create-agents.py` | Sonnet (Opus designs rubric) | A-07 | Guardrail ACTIVE + attached; payment probe blocked; planted off-KB answer rejected in test |
| A-10 | `[ ]` | Retrieval eval harness: golden questions (≥2/tree + 1/brand + mic) hit `Retrieve`, assert expected `tree_id` in top-3; real PR gate | new `tests/retrieval/`, `scripts/eval-retrieval.py`; `pr-validation.yml` | Sonnet | A-06 | ≥90% top-3 hit rate; gate fails PR below threshold |
| A-11 | `[ ]` | Plumb KB/guardrail config to Lambdas via env + SSM; scoped `bedrock:Retrieve` on the KB ARN | `template.yaml`, `cmd/lex-lambda/main.go` | Haiku | A-05 | Cold-start logs show KB params; no PLACEHOLDER fallbacks; no secrets outside SSM/env |
| DM-2…DM-5 | `[ ]` | Domain expansion docs: DECT/dongle, Teams/Zoom/Webex, firmware/driver-per-brand, audio-quality flows | `knowledge-base/**` + sidecars | Haiku author / Sonnet review | A-01,A-02 | Each in the standard format; indexed; covered by an A-10 golden question |

### WS-B — Conversation Engine (CC-2…CC-5, VX-1…VX-6)

**Design stance:** the trees are *data*, not prose to improvise over. Encode them as a Go state machine (`internal/triage`) that owns navigation; the Bedrock agent renders/explains the current step in persona voice. Session truth lives in the provisioned-but-unused `SessionTable`, keyed by Lex `sessionId` (= Connect ContactId), mirrored into Lex session attributes each turn. This fixes PRD honest-truth #4: `lex-lambda/main.go` reads `frustration_count`/`failed_steps` that nothing writes.

| ID | St | Task | Files / resources | Model | Deps | DoD |
|---|---|---|---|---|---|---|
| B-01 | `[x]` | Session store package: Load/Save `models.Session` against `SessionTable` (TTL refresh); typed accessors for current_tree, current_step, attempted_steps, failed_steps, frustration_count, pace_rate, last_response, low_asr_count, no_match_count | new `internal/session/store.go`+test; `internal/models/types.go` (`TriageState`) | Sonnet | — | Round-trip tests pass; TTL set; concurrent-turn safety via conditional write on `LastActivity` |
| B-02 | `[ ]` | Wire session store into the Lex handler (and `/chat` parity): load at top, merge to session attrs, persist after response (incl. escalation/Close) | `cmd/lex-lambda/main.go` | Sonnet | B-01 | 5-turn call: state survives in DynamoDB; chat shares the store; graceful degrade + log on read failure |
| B-03 | `[x]` | **Design the triage state machine** (CC-2): trees-as-data schema (step id, read-aloud key, yes/no/other transitions, KB doc ref, escalation terminals incl. "≥2 reboots"), top-level symptom fork, and the agent contract (who says what) | design section here; `internal/triage/types.go` | Opus | — | Every §1 branch maps to a schema instance, no "misc" escape; sign-off before B-04 |
| B-04 | `[x]` | Implement the engine + tree data: pre-flight + Trees 1–8 (terminals → reasons/RMA), symptom classifier (slot/keyword first, single Haiku `Converse` fallback) | new `internal/triage/{engine,trees,classify}.go`+tests; `cmd/lex-lambda/main.go` | Fable | B-01,B-03,A-01 | Table-driven tests walk every path; scripted Tree-8 reaches RMA in order; classifier picks correct tree on the 8 index utterances |
| B-05 | `[ ]` | "Did it work?" verify loop + step-state (CC-3): parse worked/didn't/unclear; failure appends attempted_steps, increments failed_steps, advances If-NO; success → disposition | `internal/triage/engine.go`; `internal/handlers/lex.go`; `cmd/lex-lambda/main.go` | Fable | B-04 | 3 failures → failed_steps=3 + 3 attempted entries; "yes, fixed" → resolved+Close; off-script re-prompts once then treats as no |
| B-06 | `[ ]` | Make escalation counters real (CC-4): `DetectEscalation` returns the frustration delta (currently computed then discarded); handler persists `frustration_count += delta` + engine's `failed_steps` so the ≥3/≥5 thresholds fire | `internal/handlers/escalation.go`; `cmd/lex-lambda/main.go`; new `escalation_test.go` | Sonnet | B-02 | 3 turns w/ a frustration phrase each escalates on turn 3 (`user_frustrated`); escalation session attrs present for OPS-2 |
| B-07 | `[ ]` | Lex slots (CC-5): ConnectionType / HeadsetBrand / IssueType slot types + slots + elicit/confirm/DTMF-spelling fallback; read into triage state + KB metadata filters | `template.yaml` (SlotTypes/Slots); `cmd/lex-lambda/main.go`; `internal/triage/classify.go` | Sonnet | B-04,A-02 | "My Jabra USB mic is dead" fills 3 slots in one shot; missing slots elicited ≤once; values flow into retrieval filters |
| B-08 | `[ ]` | Streaming/partial TTS (VX-1): Lex fulfillment progress updates (persona filler) + first-sentence fast-path (split `bedrock.go` stream at first sentence; cap agent length) | `internal/agents/bedrock.go`; `template.yaml` (FulfillmentUpdates); `cmd/lex-lambda/main.go` | Opus design → impl | A-07 | Real call: audible response ≤2s after end-of-speech; no correctness regression; Lex-limitation note recorded |
| B-09 | `[ ]` | Barge-in (VX-2): `allowInterrupt` on messages/prompts, audio+DTMF input specs, interruptible Connect blocks | `template.yaml` (Lex locale + `LexContactFlow`) | Haiku | — | Speaking over a step cuts TTS ~500ms; interruption transcribed as next input |
| B-10 | `[ ]` | No-input/no-match ladder (VX-3): FallbackIntent tracks no_match_count → rephrase → simplified yes/no or DTMF → offer human; no-input timeouts | `cmd/lex-lambda/main.go`; `internal/handlers/lex.go`; `template.yaml` | Sonnet | B-02 | 3 no-matches walk the ladder to escalation; counter resets on a match |
| B-11 | `[ ]` | Persona-selection IVR (VX-4): DTMF menu before the Lex block sets `persona_id` (already read by the handler; today always defaults to tangerine) | `template.yaml` `LexContactFlow` JSON | Sonnet | — | Press 2 → Joseph's voice on turn 1; timeout → default, no dead end |
| B-12 | `[ ]` | Pace + repeat/slow/loud (VX-5/AX-2): intercept pre-Bedrock; re-emit last_response at pace_rate −15%/step (floor 70%); persist; `BuildSSML` gains rate/volume override (today hardcodes persona rate) | `cmd/lex-lambda/main.go`; `internal/handlers/lex.go`; `internal/session` | Sonnet | B-02 | "Slow down" replays prior step slower; next answer stays slow; "repeat" works after any turn |
| B-13 | `[ ]` | Low-confidence ASR fallback (VX-6): track consecutive `<0.5` `TranscriptionConfidence`; at 2 → confirm + DTMF; at 3 → human w/ `escalation_reason=asr_low_confidence` (new reason) | `cmd/lex-lambda/main.go`; `internal/session`; `internal/handlers/escalation.go`; `internal/triage/engine.go` | Sonnet | B-02,B-06 | Low-confidence sequence triggers ladder at right thresholds; never fires on one garbled turn |
| B-14 | `[ ]` | Engine test suite + transcript replayer: golden multi-turn transcripts replayed in-process; blocking PR gate | new `internal/triage/engine_test.go`, `cmd/lex-lambda/main_test.go`, `tests/transcripts/*.json`; `pr-validation.yml` | Sonnet | B-04,B-05,B-06 | `go test ./...` exercises every terminal + all escalation reasons; coverage on `internal/triage` ≥80% |

### WS-C — Contact Center & Amazon Connect (OPS-1…OPS-10)

**SAM-able vs script:** `HoursOfOperation`, `Queue`, `RoutingProfile`, `User`, all `ContactFlow` types, and `PhoneNumber` have CFN types → move into `template.yaml`. Stays imperative in `setup-connect.py`/deploy.yml: looking up the CONNECT_MANAGED default security-profile ARN (pass as a stack parameter), phone-number quota retry/cleanup, phone↔flow association verification.
**As-built trap:** three competing flow definitions exist (inline in `template.yaml`; `contact-flow-lex.json` with the empty-params dead-end transfer; a third copy in `setup-connect.py`). WS-C-05 single-sources them **before** any other flow edit.

| ID | St | Task | Files / resources | Model | Deps | DoD |
|---|---|---|---|---|---|---|
| WS-C-01 | `[~]` | `AWS::Connect::HoursOfOperation` (M–F 09:00–17:00 America/New_York, default) | `template.yaml` | Haiku | — | Deploys via GHA; ARN → output + SSM `/connect/hours-arn` |
| WS-C-02 | `[~]` | Escalation `AWS::Connect::Queue` bound to hours, MaxContacts set | `template.yaml` | Haiku | C-01 | Queue ACTIVE; ARN → SSM `/connect/escalation-queue-arn` |
| WS-C-03 | `[~]` | `AWS::Connect::RoutingProfile` (escalation + skill queues, VOICE concurrency 1) | `template.yaml` | Sonnet | C-02 | Deploys; default outbound queue set |
| WS-C-04 | `[~]` | ≥1 `AWS::Connect::User` — password from GH Actions secret `CONNECT_AGENT_PASSWORD` via `NoEcho` param; security-profile ARN looked up in deploy.yml; **no Secrets Manager** | `template.yaml`, `deploy.yml` | Sonnet | C-03 | User logs into CCP; password never in logs/template |
| WS-C-05 | `[x]` | **Single-source the flows**: delete inline + `setup-connect.py` copies; `contact-flow-*.json` is the only truth, injected via `Fn::Sub` at build | `template.yaml`, `contact-flow-lex.json`, `setup-connect.py`, `deploy.yml` | Opus | — | One def per path; script no longer edits flows; deployed content matches repo byte-for-byte |
| WS-C-06 | `[~]` | **Fix the dead-end transfer (OPS-1)**: `UpdateContactTargetQueue` → `CheckHoursOfOperation` → `TransferContactToQueue` with error branches to C-09/10 | `contact-flow-lex.json`, `template.yaml` | Opus | C-02,C-05 | Live: "agent" rings the C-04 user's CCP; no silent disconnect |
| WS-C-07 | `[ ]` | **Warm-transfer context (OPS-2)**: Lambda writes escalation_reason/issue_type/steps_tried/persona_id/transcript_summary to session attrs; flow copies to contact attrs before transfer | `internal/handlers/lex.go`, `escalation.go`, `contact-flow-lex.json` | Fable | C-06 | Attributes on the contact record + agent CCP; unit test asserts them on escalation |
| WS-C-08 | `[ ]` | Agent whisper flow (`AGENT_WHISPER`) speaks issue + reason pre-connect | `template.yaml`, new `infrastructure/flow-agent-whisper.json` | Sonnet | C-07 | Agent hears the summary before bridging |
| WS-C-09 | `[ ]` | After-hours (OPS-3) + voicemail-as-text (OPS-5): closed → message → Lex dictation turn → SES email of transcript. *No KVS audio voicemail (cost/complexity, zero v1 value).* | `contact-flow-lex.json`, `internal/handlers/lex.go`, `template.yaml` (SES) | Fable | C-06 | After-hours call → closed message → email w/ transcript + hashed ANI |
| WS-C-10 | `[ ]` | Queued callback (OPS-4) when `LongestQueueWaitTime` threshold hit → set callback number (ANI) → callback queue | new `infrastructure/flow-customer-queue.json`, `template.yaml` | Fable | C-06 | Caller opts in, hangs up, agent gets a callback contact |
| WS-C-11 | `[ ]` | CSAT disconnect flow (OPS-6, infra side; content in E-7): DTMF 1–5 → Lambda writes score+disposition+path to DynamoDB | new `infrastructure/flow-disconnect-csat.json`, handler, `template.yaml` | Sonnet | C-06 | Score lands w/ contactId, path, escalated y/n |
| WS-C-12 | `[ ]` | Disposition emission (OPS-7, infra side; taxonomy in E-8): bot sets `disposition` attr; EventBridge Connect-contact-events rule → tiny Lambda persists final record | `template.yaml` (rule+Lambda), `internal/handlers/lex.go` | Fable | C-07,E-8.1 | Every contact → a disposition row; abandoned calls recorded |
| WS-C-13 | `[ ]` | Repeat-caller ANI (OPS-8): hash ANI (SHA-256 + salt from SSM SecureString) → DynamoDB lookup of prior dispositions → greet w/ context. No CRM in v1 | handler/branch, `contact-flow-lex.json`, `template.yaml` | Fable | C-12 | 2nd call from same number greeted w/ context; raw ANI never stored/logged |
| WS-C-14 | `[ ]` | Skills routing (OPS-9, Could): hardware/platform/genesys queues on the profile; branch on `issue_type` | `template.yaml`, `contact-flow-lex.json` | Sonnet | C-03,C-07 | Issue-typed calls land in the matching queue |
| WS-C-15 | `[ ]` | Supervisor monitor/barge (OPS-10, Could): enhanced monitoring + documented console steps | `template.yaml`, `docs/connect-operations.md` | Haiku | C-04 | Documented; monitor verified once manually |

### WS-D — Security, Privacy & Compliance (SEC-1…SEC-7)
**Decision (Q-B):** v1 **records + transcribes** calls. SSM Parameter Store + GH OIDC replace every secrets-manager temptation. Because recording is on, consent disclosure, two-party-consent handling, encrypted storage, retention, and Contact-Lens-driven QA (REL-6) are all in scope.

| ID | St | Task | Files / resources | Model | Deps | DoD |
|---|---|---|---|---|---|---|
| WS-D-01 | `[~]` | SEC-1 consent disclosure as the **first** audible block on every path ("this call is recorded and transcribed, and uses an AI assistant…"); signed-off recording policy doc; two-party-consent stance documented | `contact-flow-lex.json`, `docs/recording-policy.md` | Sonnet (Opus reviews policy) | C-05 | Disclosure is the first block on every path; policy doc merged; consent-state handling described |
| WS-D-01b | `[ ]` | Enable recording storage: Connect `InstanceStorageConfig` for `CALL_RECORDINGS` (+ `CHAT_TRANSCRIPTS`) → **SSE-KMS** S3 bucket with a 90-day lifecycle; KMS key is a customer-managed CMK *only here* (recording PII justifies the $1/mo; DynamoDB stays on AWS-owned keys) | `template.yaml` (S3 bucket, KMS key, InstanceStorageConfig) | Sonnet | C-05, WS-D-01 | Test call produces an encrypted recording object in S3; lifecycle expires at 90d; bucket blocks public access |
| WS-D-01c | `[ ]` | Enable **Contact Lens** on **100% of calls** (Q-H): turn on in the contact flow (`Set recording and analytics behavior`), real-time + post-call analysis; redact PII in transcripts (Contact Lens PII redaction). *Cost: ~$0.015/analyzed-min × all calls — accepted per Q-B/Q-H; sample-rate remains a future cost lever if volume grows.* Feeds REL-6 QA scorecards | `contact-flow-lex.json`, `template.yaml` | Sonnet | WS-D-01b | Contact Lens transcript + sentiment visible on every test contact; PII redacted in the stored transcript |
| WS-D-02 | `[x]` | SEC-2 structured logging + PII out of logs: `log/slog` JSON; never log raw transcripts/full events/ANI (hashed ANI + contactId only); transcript text only at DEBUG, truncated | `internal/handlers/lex.go`, `internal/agents/bedrock.go`, `cmd/*/main.go` | Fable | — | grep of test logs: no phone numbers / transcript bodies at INFO; redaction unit test |
| WS-D-03 | `[x]` | SEC-2 CloudWatch Logs `DataProtectionPolicy` (PhoneNumber/Name/Address) on both log groups | `template.yaml` | Haiku | — | Injected fake ANI appears masked |
| WS-D-04 | `[~]` | SEC-3 OIDC provider + `HeadsetDeployRole` trust-scoped to this repo/branches (+ read-only PR role); least-priv permission boundary | new `infrastructure/oidc-bootstrap.yaml`, `.github/workflows/bootstrap-oidc.yml` | Opus | — | Role assumable only from this repo; no `AdministratorAccess` |
| WS-D-05 | `[ ]` | SEC-3 swap all 3 workflows to `role-to-assume` (`id-token: write`); delete the keys gate; **deactivate the IAM user keys** | `deploy.yml`, `destroy.yml`, `pr-validation.yml` | Haiku | D-04 | Green deploy w/ zero static keys; old keys deactivated |
| WS-D-06 | `[ ]` | SEC-4 least-privilege Bedrock IAM: replace every `Resource:'*'` with explicit model/inference-profile ARNs + `InvokeAgent` scoped to the supervisor alias ARN; drop unused actions | `template.yaml` | Opus | — | Deploy green; call + /chat still work; no `bedrock:* on *` |
| WS-D-07 | `[ ]` | SEC-5 website: re-enable PublicAccessBlock, drop public policy, front with CloudFront + OAC + default cert (free tier ≈ $0) | `template.yaml`, `deploy.yml` | Sonnet | — | Direct S3 URL → 403; CloudFront URL serves over TLS |
| WS-D-08 | `[ ]` | SEC-5 lock `/chat`: shared API token in SSM SecureString, constant-time `X-Api-Key` check; CORS = CloudFront domain only; stage throttling (5 rps/burst 10). *HTTP API has no native keys; in-handler token is cheapest correct; no Secrets Manager.* | `template.yaml`, `cmd/lex-lambda/main.go`, `website/` | Fable | D-07 | No token → 401; valid token from allowed origin → 200; throttling observable |
| WS-D-09 | `[ ]` | SEC-6 retention & DSAR: session TTL ≤24h; disposition/CSAT `ttl` ~180d; `scripts/dsar-delete.py` (delete-by-hashed-ANI) via manual `workflow_dispatch`; data-inventory doc | `template.yaml`, `scripts/dsar-delete.py`, `.github/workflows/dsar.yml`, `docs/data-inventory.md` | Fable | C-12,C-13 | DSAR run removes a test caller's rows; inventory lists every store + retention |
| WS-D-10 | `[~]` | SEC-7 no-payment guardrail (overlaps A-09): denied-topic payments/card + card-pattern PII filter + system-prompt clause + refusal test | `template.yaml`/`create-agents.py`, `internal/agents/bedrock.go`, tests | Sonnet | — | "Take my card number" → refusal; test green |
| WS-D-11 | `[ ]` | STRIDE threat-model review of the finished WS-C/WS-D surface + accepted-risk register | `docs/threat-review.md` | Opus | D-01…10 | Every finding has a task or an explicit accepted-risk entry |

### WS-E — Accessibility, Inclusion & CX Quality (AX-1…AX-6, OPS-6, OPS-7, KPIs)
**Posture:** an audio-only audio-diagnosis bot structurally excludes the people most likely to have audio problems. AX-1 is the highest inclusion risk and sequences first.

| ID | St | Task | Files / resources | Model | Deps | DoD |
|---|---|---|---|---|---|---|
| E-1.1 | `[ ]` | Channel-switch contract: a session-scoped snapshot any channel can render | `internal/session/`, `internal/handlers/lex.go` | Opus | CC-4 | One `SessionSnapshot` written each turn; renders identically as voice SSML / chat / SMS (unit test) |
| E-1.2 | `[ ]` | "Text me these steps" via AWS End User Messaging SMS (toll-free already claimed); step-formatter; opt-in/STOP compliance | new `internal/notify/sms.go`, `template.yaml` (IAM + SSM `/sms-origination-arn`), `cmd/lex-lambda/main.go` | Fable | E-1.1 | "Text me the steps"/DTMF 9 confirms mobile via ANI, sends ordered numbered SMS, honors STOP, logs delivery |
| E-1.3 | `[ ]` | Publish WCAG 2.1 AA web-chat UI wired to `ChatApiUrl`, sharing session via snapshot | new `website/{index.html,chat.js,chat.css}`, `deploy.yml`, `template.yaml` | Sonnet | E-1.1; SEC-5 | Deploys via pipeline; keyboard-only, NVDA-tested, ARIA live region; round-trips w/ persona+language |
| E-1.4 | `[ ]` | "Email me the steps" (SES, HTML+text) from chat + voice (voice offers email only if SMS declined) | new `internal/notify/email.go`, `template.yaml` (SES), SSM `/steps-from-email` | Sonnet | E-1.1,E-1.2 | Email correctly formatted; bounce handling logged |
| E-1.5 | `[ ]` | Early text-fallback offer in the flow ("press 9 for help by text"); silent-call path auto-offers SMS before disconnect | `contact-flow-*.json`, Lex no-input config | Sonnet | E-1.2,VX-3 | Press-9 from any prompt; silent caller gets SMS offer within 2 no-input cycles |
| E-1.6 | `[ ]` | Channel-switch continuity test (voice→SMS/chat, no step repeated/skipped) | `tests/conversations/channel_switch_test.go` | Sonnet | E-1.1–1.5,E-9.2 | SMS after Tree-1 step 3 shows 1–3 done, 4 next; runs in CI |
| E-2.1 | `[ ]` | Repeat/Slower/Louder intents (+ synonyms) intercepted before Bedrock | `template.yaml` (Lex), `internal/handlers/lex.go` | Sonnet | — | Resolve locally <500ms; "repeat" re-emits cached last turn |
| E-2.2 | `[ ]` | Persist pace/volume per session; apply to all later SSML (100→85→70%; medium→loud), overriding persona prosody | `internal/handlers/lex.go` `BuildSSML()`, `internal/session/`, `personas/*.json` | Sonnet | CC-4,E-2.1 | "Slow down" once → slower for the whole call; survives turns; persona×override tests |
| E-2.3 | `[ ]` | DTMF equivalents (7=repeat,8=slower,9=text) announced in greeting/help | `contact-flow-*.json`, `internal/handlers/lex.go`, `personas/*.json` | Haiku | E-2.1,E-1.5 | Each key matches its voice command; greeting copy updated |
| E-3.1 | `[ ]` | `personas/clear.json`: neutral en-US neural, rate 85%, no idioms, one instruction/sentence, repeats key action | new `personas/clear.json` | Fable | — | Validates against loader; ≤15-word sentences, grade-6 reading level |
| E-3.2 | `[ ]` | Plain-language lint of all phrase banks (idiom blocklist); fix `clear.json` | `personas/*.json`, new `scripts/lint-personas.py` | Haiku | E-3.1 | Lint runs in PR validation; fails on idioms in `clear.json` |
| E-3.3 | `[ ]` | Wire `clear` into seeding + IVR; auto-fallback persona after 2 ASR low-confidence/no-match | `deploy.yml`, `contact-flow-*.json`, `internal/handlers/lex.go` | Sonnet | VX-4,VX-6,E-3.1 | Menu option 4 = "clear and slow"; low-confidence switches + announces; logged as metric dim |
| E-4.* | `[ ]` | **Spanish + multi-language — DEFERRED to M4/Later per Q-E (English-only at launch).** When pursued: language-select IVR ("para español, oprima dos") threading a `language` attribute everywhere; Lex `es_US` locale (idiomatic intents/utterances incl. pace + escalation); Spanish persona variants (`*-es`, incl. `clear-es`, locale-aware loader key); Spanish KB (`knowledge-base/es/**`) with dual-language Windows menu paths; an add-a-language checklist proving it's config+content only (no Go changes). **Design now keeps this cheap later:** any task that emits caller-facing copy or SSML must route through a single language-aware path (don't hardcode en-US) so adding a language is content, not refactoring. | `contact-flow-*.json`, `template.yaml` (BotLocale), `personas/*-es`, `internal/persona`, `knowledge-base/es/**` | Sonnet/Fable/Opus when activated | CC-4, CC-1 | Deferred; the only M-scope obligation now is that E-1/E-2/E-8 thread a `language` attribute and never hardcode en-US, so Spanish can be added later without Go changes |
| E-5.1 | `[ ]` | Sensory-cue inventory: tag every success-check (visual/auditory/tactile/screen) — ~35 in `troubleshooting.md` | `knowledge-base/_audit.md` | Haiku | — | Every cue catalogued w/ modality + proposed alternative |
| E-5.2 | `[ ]` | Rewrite content so every success-check has a non-visual AND non-auditory alternative | `troubleshooting.md`, `knowledge-base/**` | Sonnet | E-5.1,CC-1 | No check relies on one modality; KB re-synced |
| E-5.3 | `[ ]` | Agent rule: on "can't see/hear that" (or SMS/relay path), proactively offer the alternative check | persona prompts, A-09 guardrail | Sonnet | E-5.2,CC-6 | "Blind caller checks mute" + "Deaf caller" golden cases pass |
| E-6.1 | `[ ]` | Relay-tolerant timing profile (extended timeouts, relaxed reprompts) on relay declaration | `template.yaml` (Lex timeouts), `contact-flow-*.json`, `internal/handlers/lex.go` | Sonnet | E-1 | "Relay call" sets flag: timeouts ×3, persona→clear, pace 70%, addresses the caller |
| E-6.2 | `[ ]` | Relay-aware prompting: short typeable sentences, ≤1 action/turn when relay flag set | persona prompts, `internal/handlers/lex.go` | Fable | E-6.1 | Relay golden conversation passes |
| E-6.3 | `[ ]` | TTY position + accessibility statement page (711 relay + prominent SMS/chat) | `website/accessibility.html`, decision record here | Haiku | E-1.3,E-6.1 | Statement live listing all access channels |
| E-7.1 | `[ ]` | Post-call DTMF CSAT (1–5) + FCR question; survey config in SSM | `contact-flow-*`, handler, SSM `/csat-config` | Sonnet | OPS-1/2,E-8.1 | Bot-only + escalated calls surveyed; stored w/ contactId/path/persona/lang/disposition; hang-up = null not 0 |
| E-7.2 | `[ ]` | Chat thumbs-up/down + end-of-chat 1–5, same schema | `website/chat.js`, `/chat` handler | Sonnet | E-1.3 | Accessible buttons; events keyed by sessionId+turn |
| E-7.3 | `[ ]` | SMS CSAT for text-channel callers (reply 1–5) + ingestion | `internal/notify/sms.go`, inbound-SMS handler, `template.yaml` | Sonnet | E-1.2,E-7.1 | Reply parsed/stored; one re-ask then stop; STOP honored |
| E-7.4 | `[ ]` | CSAT storage (`SurveyTable` TTL) + EMF metrics `CSAT{channel,path,persona,language,disposition,escalated}` | `internal/metrics/`, `template.yaml` | Sonnet | E-7.1–7.3 | Dashboard shows CSAT bot-only vs escalated; EMF unit tests |
| E-8.1 | `[ ]` | **Disposition taxonomy** (one terminal code/contact): contained_resolved/contained_unresolved/escalated_user_requested/escalated_user_frustrated/escalated_troubleshooting_exhausted/abandoned/transferred_to_text_channel/after_hours_deflected | this file + `internal/disposition/disposition.go` | Opus | CC-3, escalation.go reasons | Enum + precedence documented; every ending maps to one code; Product-reviewed |
| E-8.2 | `[ ]` | Emit disposition everywhere (contact attr + SessionTable + metric{path,persona,language,issue_type}) | `internal/handlers/lex.go`, `contact-flow-*`, `internal/metrics/` | Sonnet | E-8.1,CC-4 | 100% of test contacts end with a disposition metric; "unknown disposition > 2%" alarm |
| E-8.3 | `[ ]` | Containment/FCR rollup widgets (containment, escalation-by-reason, per-path A/B) | dashboard JSON in `template.yaml` | Haiku | E-8.2,REL-2 | Dashboard renders PRD §6 north-star from real data |
| E-9.1 | `[ ]` | Author 11 golden conversation YAMLs (per-turn assertions, see §3.E note) | `tests/conversations/*.yaml` | Fable | troubleshooting.md, E-8.1 | All validate; each maps to a tree/section + terminal disposition |
| E-9.2 | `[ ]` | Conversation test harness replaying YAMLs vs `/chat` + Lex `RecognizeText`; gated post-deploy smoke | `tests/conversations/driver_test.go`, `deploy.yml` | Fable | E-9.1,REL-1 | Runs in pipeline, fails on assertion; per-case latency; flake retry=1 |
| E-9.3 | `[ ]` | LLM-as-judge rubric (groundedness/step-order/register/empathy/no-fabrication); judge=Haiku, Opus designs + 10% weekly audit | `tests/judge/` | Opus design, Haiku runtime | E-9.2,CC-1 | Judge ≥0.8 vs 30-conv human-labeled set; groundedness gate fails CI for goldens |
| E-9.4 | `[ ]` | Accessibility acceptance pass (SMS E2E, NVDA, relay sim, Spanish IVR, pace) before each phase exit | `tests/acceptance/ax-checklist.md` | Sonnet author + human | E-1…E-6 | Checklist run + attached to release; AX regression blocks sign-off |
| E-9.5 | `[ ]` | Capture gap events (no retrieval / "I don't know" / 2× low-confidence / unresolved-or-exhausted) to `GapEvents` (TTL 90d), **redacted** | `internal/analytics/gaps.go`, `template.yaml` | Sonnet | CC-1, SEC-2 (must land first) | Gap event for `out_of_scope_bluetooth_macos`; zero raw PII |
| E-9.6 | `[ ]` | Weekly gap report (cron GHA): cluster last week's gaps, open/update a GitHub issue w/ top 10 + samples | `.github/workflows/gap-report.yml`, `scripts/gap-report.py` | Fable | E-9.5 | Mondays; ranked gaps w/ counts/languages + proposed disposition; runs on OIDC role |
| E-9.7 | `[ ]` | Close the loop: each gap issue ends in a content PR, a new golden test, or an explicit out-of-scope note; track recurrence | process doc here, repo labels | Haiku | E-9.6 | Process documented; content-fixable gaps get a regression golden test |

> **§3.E note — golden conversation corpus (`tests/conversations/`):** versioned multi-turn YAMLs with per-turn assertions (intent, KB doc cited, step order, persona register, terminal disposition, latency budget), derived from the trees so the corpus *is* the content acceptance contract. Initial 11: `usb_no_audio_resolved`, `mic_privacy_resolved`, `not_detected_rma_escalation`, `explicit_escalation`, `frustrated_caller`, `spanish_caller_mic_fix`, `deaf_caller_sms_fallback`, `relay_call_clear_pace`, `repeat_slow_down`, `mute_sync_privacy_warning`, `out_of_scope_bluetooth_macos`.

### WS-F — Observability & Cost (REL-2, REL-7)

| ID | St | Task | Files / resources | Model | Deps | DoD |
|---|---|---|---|---|---|---|
| WS-F-01 | `[x]` | SNS `headset-alerts-prod` + email sub (proffitt.jeremy@gmail.com), $0 | `template.yaml` | Haiku | — | Sub confirmed; test publish received |
| WS-F-02 | `[x]` | Core alarms → SNS: per-Lambda Errors/Duration-p95>20s/Throttles; Bedrock ThrottledCount/ServerErrors by ModelId (~$1/mo) | `template.yaml` | Sonnet | F-01 | Forced error emails within 5 min |
| WS-F-03 | `[ ]` | Connect alarms: MissedCalls>0, LongestQueueWaitTime>300s, concurrency-quota breaches | `template.yaml` | Sonnet | C-02,F-01 | Synthetic unanswered call trips MissedCalls |
| WS-F-04 | `[~]` | **Config-drift canary** (hourly EventBridge micro-Lambda): metrics `AgentIdIsPlaceholder`/`PhoneNumberPending` from SSM + alarms (~$0.60/mo) — turns today's silent PLACEHOLDER warning into a page | new `cmd/canary-lambda/`, `template.yaml` | Fable | F-01 | Manually setting a param to PLACEHOLDER alarms within 2h |
| WS-F-05 | `[ ]` | CloudWatch dashboard (Lambda/Bedrock/Connect + CSAT/disposition widgets), $0 | `template.yaml` | Haiku | F-02…04 | All widgets render w/ live data after a test call |
| WS-F-06 | `[ ]` | X-Ray `Tracing: Active` + Go SDK middleware around Bedrock/DynamoDB clients (free tier) | `template.yaml`, `internal/agents/bedrock.go`, `cmd/*/main.go` | Sonnet | — | Service map shows Lambda→Bedrock→DynamoDB |
| WS-F-07 | `[ ]` | AWS Budgets (monthly $50, alerts 50/80/100%) + Cost Anomaly monitor → email ($0) | `template.yaml` | Sonnet | — | Budget visible; threshold email tested |
| WS-F-08 | `[ ]` | Lambda reserved concurrency (Lex 10, Nova 2, canary 1) blast-radius cap ($0) | `template.yaml` | Haiku | — | Beyond cap throttles instead of running away |
| WS-F-09 | `[ ]` | Bedrock token caps (`maxTokens` on agent + direct invokes); confirm sub-agents actually use Haiku | `internal/agents/bedrock.go`, `scripts/create-agents.py` | Sonnet | — | Long response capped; config unit test |
| WS-F-10 | `[ ]` | Orphan-number sweeper (weekly cron, OIDC): list numbers, flag unreferenced/unassociated, release after grace; also fix destroy.yml's no-op release | new `.github/workflows/sweep-numbers.yml`, `scripts/sweep-numbers.py`, `destroy.yml` | Fable | D-05 | Seeded orphan detected + released; destroy.yml actually releases |

### WS-G — Test & CI/CD + Reliability (REL-1, REL-3, REL-4, REL-5, SEC-3)

| ID | St | Task | Files / resources | Model | Deps | DoD |
|---|---|---|---|---|---|---|
| WS-G-01 | `[x]` | Unit tests for `escalation.go` (keyword triggers, reason classification, a test exposing the never-incremented counters) | new `internal/handlers/escalation_test.go` | Fable | — | ≥90% pkg coverage; counter bug captured |
| WS-G-02 | `[x]` | Unit tests for `lex.go` (event parse, SSML/HTML-injection escaping, `/chat` branch, session passthrough) | new `internal/handlers/lex_test.go` | Sonnet | — | ≥80% pkg coverage; injection cases green |
| WS-G-03 | `[x]` | Unit tests for `bedrock.go` (mock agent-runtime: streaming assembly, 25s timeout, error mapping) | `internal/agents/bedrock.go` (+iface), new `bedrock_test.go` | Sonnet | — | ≥75% pkg coverage; timeout path tested |
| WS-G-04 | `[x]` | Unit tests for `persona.go` (hit/miss → `DefaultPersona()`, malformed item) | new `internal/persona/persona_test.go` | Sonnet | — | ≥85% pkg coverage |
| WS-G-05 | `[~]` | **Honest CI**: remove `continue-on-error`/`\|\| true`; make integration tests real (call /chat w/ token, assert non-placeholder answer) and fail the deploy on failure | `deploy.yml`, `tests/integration/` | Sonnet | G-01…04, D-08 | Broken deploy goes red; green = real /chat round-trip |
| WS-G-06 | `[x]` | Real coverage gate + kill the hardcoded ✅ summary (read actual `needs.*.result`; fail <60%, ratchet later) | `pr-validation.yml` | Haiku | G-01…04 | Coverage-dropping PR blocked; summary shows ❌ on failure |
| WS-G-07 | `[ ]` | DLQ + idempotency (REL-3): SQS DLQ + `EventInvokeConfig` on async; **idempotency** via conditional-write turn records keyed `contactId#turn` so retried invokes don't double-call Bedrock | `template.yaml`, `internal/handlers/lex.go`, tests | Fable | F-01 | Replayed event returns cached result, no 2nd Bedrock call; DLQ alarm wired |
| WS-G-08 | `[ ]` | Idempotent provisioning: `create-agents.py` + `setup-connect.py` find-before-create everywhere | `scripts/create-agents.py`, `scripts/setup-connect.py` | Fable | C-05 | Running the pipeline twice yields zero new resources, green both times |
| WS-G-09 | `[ ]` | **Nova Sonic safety gate (REL-4, revised per Q-D — completing not fencing)**: Path B stays behind `DeployNovaSonicPath` (default `"false"` until WS-H acceptance passes); the Nova contact flow transfers to the Lex flow whenever the path is disabled, so a half-built Path B can never serve live callers. Flip the default to `"true"` only when WS-H-06 is `[x]` | `template.yaml`, `deploy.yml`, `contact-flow-nova-sonic.json`, `setup-connect.py` | Fable | C-05, WS-H-06 | While disabled, calling the Nova number reaches Path A; flipping the flag after WS-H acceptance serves real Nova Sonic audio |
| WS-G-10 | `[x]` | Post-deploy smoke test (assert non-error, non-placeholder Lex response; fail deploy otherwise) | `deploy.yml` validate job | Sonnet | G-05 | Smoke exits non-zero on PLACEHOLDER/invoke error |
| WS-G-11 | `[ ]` | REL-5 — **DR runbook only (no dev/staging env per Q-G; production-only).** Write `docs/dr-runbook.md`: RTO/RPO statement, DynamoDB PITR restore steps for PersonaTable, full redeploy-from-pipeline procedure, the SSM-param re-seed list, and the recording/KB bucket recovery notes. The ephemeral-dev-stack task is dropped. Because there is no pre-prod, the M0 safety net (real tests + coverage gate + alarms + post-deploy smoke WS-G-10) is the *only* guard before prod — keep PRs small | new `docs/dr-runbook.md` | Fable | G-05 | Runbook merged and walked through once on paper (restore + redeploy + re-seed); no second environment created |

### WS-H — Nova Sonic Path B (Bidirectional Streaming Voice) — REL-4, PRD Phase-4-pulled-forward

**Why this exists:** Per Q-D the user chose to **complete** Nova Sonic rather than fence it. Today `cmd/nova-sonic-lambda/main.go` decodes audio then returns a placeholder (`// TODO: Implement actual Nova Sonic bidirectional streaming`). This workstream builds the real speech-to-speech path: caller audio → Bedrock `InvokeModelWithBidirectionalStream` (model `amazon.nova-sonic-v1:0`) → caller audio, integrated with Amazon Connect.

**Architectural reality (drives WS-H-01):** Nova Sonic needs a *persistent, low-latency, bidirectional* audio session. AWS Lambda is a poor fit for a long-lived duplex media stream; the likely target is **ECS Fargate** (a long-running streaming service) consuming Connect audio via **Kinesis Video Streams** (Connect "Start media streaming") and bridging audio back. Getting synthesized audio *back into* a live Connect call is the hard, least-mature part and must be proven in the WS-H-01 spike before the rest is built. This is the highest-risk workstream in the plan — it stays behind the WS-G-09 safety gate until WS-H-06 passes.

| ID | St | Task | Files / resources | Model | Deps | DoD |
|---|---|---|---|---|---|---|
| WS-H-01 | `[ ]` | **Design spike (go/no-go on the bridge):** prove how synthesized audio returns into a live Connect call. Evaluate Connect media-streaming (KVS) + a Fargate audio bridge vs. Connect "external voice" options; pick compute (Fargate vs Lambda); decide the audio bridge; document latency budget (<2s target) and a fallback (revert caller to Path A on stream failure) | `docs/nova-sonic-architecture.md`, decision recorded in §5 | Opus | A-07 (shared agent/KB), Q-D | Working spike: a real call hears Nova-Sonic-generated audio at least once end-to-end; architecture + cost + latency documented; fallback path defined |
| WS-H-02 | `[ ]` | KVS ingestion: Connect contact flow "Start media streaming" block; consumer reads customer-audio fragments and decodes to PCM for Bedrock | `contact-flow-nova-sonic.json`, new streaming service (`cmd/nova-sonic-stream/` or Fargate `service/`), `template.yaml` (KVS perms, ECS if chosen) | Fable | WS-H-01 | Live call's caller audio arrives as PCM frames in the service logs at real-time rate |
| WS-H-03 | `[ ]` | Bidirectional Bedrock client: `InvokeModelWithBidirectionalStream` with Nova Sonic event protocol (session start, prompt/persona config, audio-in events, audio-out + text events); inject persona voice + the same KB/agent grounding contract as Path A | new streaming client in the service; `internal/agents/` (shared types); reads persona via `internal/persona` | Opus design → Fable impl | WS-H-02, A-08 | Service streams caller audio to Nova Sonic and receives audio + transcript events; persona voice audibly applied |
| WS-H-04 | `[ ]` | Audio playback bridge: stream Nova Sonic audio-out back into the live Connect call per the WS-H-01 design; barge-in/interrupt handling (stop playback when caller speaks) | streaming service, `contact-flow-nova-sonic.json` | Fable | WS-H-01, WS-H-03 | Caller hears Nova Sonic responses in near-real-time; speaking over the bot interrupts playback |
| WS-H-05 | `[ ]` | Persona + voice mapping for Nova Sonic (map the 4 personas incl. `clear` to Nova Sonic voice ids; reconcile with `personas/*.json` `nova_sonic_voice_id`); session memory + disposition parity with Path B (write to `SessionTable`, emit disposition like CC-4/E-8) | `personas/*.json`, streaming service, `internal/session/`, `internal/disposition/` | Fable | WS-H-03, B-01, E-8.1 | Each persona produces a distinct Nova Sonic voice; Path B contacts get session records + dispositions like Path A |
| WS-H-06 | `[ ]` | **Acceptance + the gate flip:** golden voice scenarios (no-audio, mic, escalation) run on Path B; measure latency vs the <2s target and vs Path A KPIs (per-path A/B from WS-F); on pass, flip `DeployNovaSonicPath` default to `"true"` (with WS-G-09) and claim/activate the Nova number | `tests/conversations/` (voice variants), `deploy.yml`, `template.yaml`, `docs/nova-sonic-architecture.md` (results) | Sonnet | WS-H-04, WS-H-05, WS-G-09, WS-F | Path B passes the golden voice scenarios; latency report recorded; flag flipped so live callers reach real Nova Sonic; rollback-to-Path-A on stream failure verified |

> **WS-H cost note:** ECS Fargate for a streaming service is the main new cost (a small always-on task ≈ $10–15/mo, or scale-to-zero with on-demand session start if the design allows) plus Nova Sonic per-audio-minute pricing and KVS ingestion. This is the single largest cost driver introduced by the Q-D decision; WS-F-07 budgets must account for it, and WS-H-01 should evaluate scale-to-zero to keep idle cost near $0.

---

## 4. Working Logs

Append entries under the relevant milestone using this template (copy per entry):
```markdown
---
**[YYYY-MM-DD HH:MM] <agent> — <task ID> — <marker change, e.g. [ ]→[~]>**
- Decision: chosen approach + rejected alternatives & why.
- Files touched: repo paths, one per line.
- Commands run: exact build/test/gh commands. Pipeline result in one line.
- Gotchas: anything the next agent must know (wrong-account trap, flaky resource, ordering).
- Unblocked: task IDs this completion unblocks.
```

### M0 Working Log

---
**[2026-06-12] Claude(Opus orchestrator) — Batch 1: test foundation + honest CI + KB split + triage design — markers set**
- Scope: WS-A-01 [x], B-03 [x], WS-G-01/02/03/04 [x], WS-G-06 [x], WS-G-10 [x], WS-G-05 [~].
- **WS-A-01** (Sonnet subagent): split `troubleshooting.md` + 5 existing docs into 32 retrieval-sized docs under `knowledge-base/{trees,windows,brands,genesys,common}/`. Branch text copied verbatim; produced a full coverage/diff-audit table (every §1 tree, §2.1–2.8, §3 brands, §4 Genesys topic mapped). §5 (bibliography of URLs) intentionally not extracted (cited inline). Existing 5 docs retained (no contradictions; FAQ Poly entry flagged for future revision). Largest doc 767 words (≤1500-token target met).
- **B-03** (Opus subagent): authored `internal/triage/types.go` (compiles + vets clean) and `docs/triage-design.md` (B-04 sign-off artifact). Schema = trees-as-data: `SymptomFork`/`Tree`/`Step`/`Transition`/`Terminal`/`TriageState`; 8 symptom classes + preflight, no catch-all; 3 shared escalation reasons reused verbatim from escalation.go + 7 tree-driven; `RebootLimit=2`/`FailedStepsEscalationThreshold=5`/`FrustrationEscalationThreshold=3`. Agent contract: engine owns navigation (emits `StepView`), agent renders persona voice + KB-grounded answers (returns `TurnResult{worked|didnt_work|unclear}`). Note for B-04: n-way characterization forks (Trees 5/6/7) modeled as 3-transition steps; optional `Branches map` left as B-04's call.
- **WS-G-01/02** (Sonnet subagent): `internal/handlers/{escalation_test.go,lex_test.go}` → package `handlers` **100.0%** coverage. Counter bug captured in `TestDetectEscalation_CounterBugNeverIncrements` (passing, asserts current buggy behavior; comment tells B-06 to update it). SSML HTML-injection escaping tested.
- **WS-G-03** (Sonnet subagent): introduced `agentRuntimeAPI` interface in `bedrock.go` (concrete client satisfies it; `NewBedrockClient` unchanged) + extracted `processResponseStream`. Tests cover pure funcs + error + deadline-exceeded mapping → **84.6%** (agents). Streaming success path genuinely unmockable (SDK sealed event stream) — documented, not faked.
- **WS-G-04** (Sonnet subagent): introduced `dynamoAPI` interface in `persona.go` + tests for Load hit/miss/error/malformed + Save → **89.5%** (persona).
- **WS-G-06** (me): `pr-validation.yml` — added a real internal-package coverage gate (fail <60%; current 92.0%) in the `test` job; rewrote `summary` job to read `needs.*.result` (✅/❌ from reality) with `if: always()` and `exit 1` on any non-success. Killed the hardcoded ✅ table.
- **WS-G-10** (me): `tests/integration/test_smoke.py` — boto3 invoke of the deployed Lex Lambda health-check path (no agent dependency → deterministic); asserts "Health check successful". `deploy.yml` `validate-deployment` inline smoke hardened to `exit 1` on missing health-check (removed `|| echo` swallow).
- **WS-G-05** (me, **[~] partial**): removed `continue-on-error: true` and `|| true` from `deploy.yml` integration-tests (now blocking + real); added the internal-coverage gate to the deploy `build` job. **Remaining (blocked-by D-08, M3):** the "/chat with API token, assert non-placeholder grounded answer" round-trip — needs WS-D-08's shared token + live agent/KB. `setup-connect` job kept `continue-on-error` for now (blocked-by WS-G-08 idempotency; making it blocking before find-before-create risks bricking deploys on phone-number quota).
- Also gofmt-cleaned 3 pre-existing files (`cmd/lex-lambda/main.go`, `cmd/nova-sonic-lambda/main.go`, `internal/models/types.go`) that were never caught (pushes go straight to main; gofmt lint only runs on PRs). Whole module now gofmt-clean.
- Files touched: knowledge-base/** (32 new), internal/triage/types.go, docs/triage-design.md, internal/handlers/{escalation_test,lex_test}.go, internal/agents/{bedrock.go,bedrock_test.go}, internal/persona/{persona.go,persona_test.go}, tests/integration/test_smoke.py, .github/workflows/{pr-validation.yml,deploy.yml}, cmd/*/main.go + internal/models/types.go (gofmt only).
- Commands: `go build ./...` OK; `go vet ./...` OK; `go test ./...` PASS; internal-only coverage **92.0%** (gate 60%). YAML + python syntax validated locally.
- Gotchas: (1) Windows `core.autocrlf=true` — `gofmt -l` flags every file locally due to CRLF; verify content-formatting by `tr -d '\r' | gofmt -l`. (2) cmd/* main packages have 0% unit coverage by design → coverage gate scoped to `./internal/...`, documented in both workflows. (3) Pipeline verification pending (push next) — CI-task [x] markers reflect locally-verified logic; will confirm green.
- Unblocked: B-04 (triage impl — design+types ready), A-02/A-03/DM-* (KB docs exist to add sidecars/expand), all WS unit-test gate now real.
- **Pipeline result:** run 27410217843 (commit 9b15c1e) — **GREEN**. Confirmed in CI: Build&Test passed (coverage gate works), Integration Tests passed (new BLOCKING health-check smoke ran against the real deployed Lambda), Deploy SAM success, setup-connect success.

---
**[2026-06-12] Claude(Opus orchestrator) — Batch 2: PII logging, no-payment, observability, OIDC authoring**
- Scope: WS-D-02 [x], WS-D-03 [x], WS-F-01 [x], WS-F-02 [x], WS-D-04 [~], WS-D-10 [~].
- **WS-D-02** (Sonnet subagent): new `internal/logging` package — slog JSON handler (level from `LOG_LEVEL`), `HashANI` (salted SHA-256, salt from `ANI_HASH_SALT` env; WARN-once if unset), `Truncate`. Refactored cmd/lex-lambda, cmd/nova-sonic-lambda, internal/handlers/lex.go, internal/agents/bedrock.go off `log.Printf` → slog: transcripts only at DEBUG (truncated), ANI hashed, never raw at INFO. `internal/logging/logging_test.go` includes a planted-fake-ANI test asserting no PII at INFO. All tests green.
- **WS-D-10** (Sonnet subagent, **[~]**): new `internal/handlers/payment.go` — `DetectPaymentSolicitation` (14 payment phrases + card-number regex / 13–16 digit run; conservative to avoid USB/model-number false positives) + `BuildPaymentRefusalResponse` (sets `payment_blocked`, never echoes digits). Wired into both handleLexRequest + handleAPIRequest BEFORE escalation/Bedrock. No-payment clause added to DefaultPersona + all 3 personas/*.json system prompts. 27 tests, green. **Pending (A-09):** the Bedrock Guardrail denied-topic layer (belt-and-suspenders).
- **WS-D-03** (me): added CloudWatch Logs `DataProtectionPolicy` (Audit + Deidentify/mask for PhoneNumber-US/Name/Address) to both Lambda log groups. Live-mask behavior is AWS-managed; structural correctness verified.
- **WS-F-01** (me): `AlertTopic` SNS `headset-alerts-prod` + email subscription to proffitt.jeremy@gmail.com. **NOTE: recipient must click the one-time SNS confirmation email** before alarm emails deliver. `AlertTopicArn` exported for WS-F-03/04/canary.
- **WS-F-02** (me): alarms → AlertTopic: Lex Errors/Duration-p95(>20s)/Throttles; Nova Errors/Throttles; Bedrock supervisor InvocationServerErrors/InvocationThrottles by ModelId. All `TreatMissingData: notBreaching`.
- **WS-D-04** (Opus subagent, **[~]**): authored `infrastructure/oidc-bootstrap.yaml` (OIDC provider + `HeadsetDeployRole` scoped to repo+main/release refs, `HeadsetPRRole` for pull_request, least-priv boundary, NO AdministratorAccess) + `.github/workflows/bootstrap-oidc.yml` (workflow_dispatch). Structurally validated. **DEPLOY DEFERRED:** bootstrap + WS-D-05 swap + static-key deactivation is a destructive cutover needing the role's least-priv verified against the real deploy via pipeline logs — its own monitored step. Files inert until bootstrap runs.
- Files touched: internal/logging/*, internal/handlers/payment*.go, cmd/*/main.go, internal/handlers/lex.go, internal/agents/bedrock.go, internal/persona/persona.go, personas/*.json, infrastructure/template.yaml, infrastructure/oidc-bootstrap.yaml, .github/workflows/bootstrap-oidc.yml.
- Commands: `go build/vet/test ./...` green; template structurally validated (35 resources); persona JSON valid; gofmt-clean.
- Gotchas: SNS email needs manual confirm; Bedrock alarm ModelId dimension unverified (notBreaching); OIDC cutover is the single riskiest M0 action — not done blind.
- **Pipeline result:** run 27410992661 (commit 345070b) — **GREEN** (all 12 jobs success incl. Deploy SAM with the DataProtectionPolicy + alarms, and Validate Deployment).

---
**[2026-06-12] Claude(Opus orchestrator) — Batch 3: Connect escalation cluster + consent + canary**
- Scope: WS-C-05 [x]; WS-C-01/02/03/04/06 [~]; WS-D-01 [~]; WS-F-04 [~] (deploy/phone verification pending). User confirmed approach: "build + deploy, verify by phone"; user SET the `CONNECT_AGENT_PASSWORD` GH secret.
- **WS-C-05** (Opus subagent) [x]: single-sourced the Lex flow onto the **CFN-managed inline `LexContactFlow`** (decision overrides PLAN's "JSON-injected" wording — same DoD: one def/path, script no longer edits flows). Removed dead `create_lex_contact_flow`/`create_nova_sonic_contact_flow` from setup-connect.py (never called from main(); were a 2nd live definition); main() still only reads flow IDs by name + claims/associates phone numbers. `contact-flow-lex.json` demoted to non-authoritative reference via `_NOTE` key.
- **WS-C-01/02/03** [~]: `HeadsetHoursOfOperation` (M–F 09:00–17:00 America/New_York), `HeadsetEscalationQueue` (bound to hours, MaxContacts 50, SSM /connect/escalation-queue-arn), `HeadsetRoutingProfile` (VOICE concurrency 1, escalation queue as default+config). Outputs + SSM params added. → [x] on deploy green.
- **WS-C-04** [~]: `HeadsetAgentUser` (Condition `CreateAgentUser` = ConnectAgentPassword + AgentSecurityProfileId both non-empty), SOFT_PHONE, username headset.agent. deploy.yml SAM step now passes `ConnectAgentPassword` (from the now-set secret, never echoed) + looks up the CONNECT_MANAGED "Agent" security-profile Id. → [x] after CCP login verification.
- **WS-C-06** [~]: fixed the dead-end transfer (OPS-1). Flow escalation path: `checkHours` (CheckHoursOfOperation) → open: `setQueue` (UpdateContactTargetQueue=escalation queue) → `transferToQueue` (TransferContactToQueue) → errors → graceful `agentsBusy`; closed → `afterHours` message → disconnect. No empty-param transfer, no silent disconnect. → [x] after phone verification ("agent" rings the CCP).
- **WS-D-01** [~]: recording/AI **consent disclosure is the StartAction (first block)** of the Lex flow; `docs/recording-policy.md` created (90d retention, disclosure→implied-consent stance). Pending: same consent on the Nova flow + recording STORAGE (WS-D-01b KMS bucket/InstanceStorageConfig + WS-D-01c Contact Lens).
- **WS-F-04** [~]: config-drift canary (`cmd/canary-lambda`, hourly EventBridge, metrics AgentIdIsPlaceholder/PhoneNumberPending in HeadsetAgent/Canary ns, 2 alarms → AlertTopic, reserved concurrency 1). arm64 build + full module green. → [x] on deploy green.
- Validation: template structurally valid (CFN-aware loader); flow Content valid JSON, StartAction=consent, 17 actions, zero dangling transitions, all real Connect action types; setup-connect.py py_compile OK; deploy.yml YAML OK + password never echoed; `go build/vet/test ./...` green; gofmt clean.
- **Residual risk (flagged for the phone test):** `CheckHoursOfOperation` param key (`HoursOfOperationId`=ARN) + condition operand (`["true"]`), and `UpdateContactTargetQueue` (`QueueId`=ARN) are the documented forms but only fully validate at CFN deploy / on a live call. If the flow content is rejected, CFN rolls back to the prior working flow (no permanent breakage) and we iterate from the error.
- Files: infrastructure/template.yaml, scripts/setup-connect.py, .github/workflows/deploy.yml, infrastructure/contact-flow-lex.json, docs/recording-policy.md, cmd/canary-lambda/* (+ go.mod/go.sum cloudwatch dep).
- **Remediation (batch-3 took 3 deploy attempts; each rolled back cleanly — no live breakage):**
  1. Attempt 1 (commit 01369a2) FAILED at SAM Build: the canary `Makefile` fell to `go build .` which can't find go.mod in SAM's CodeUri sandbox. Fix (b3fd1f7): deploy.yml now pre-builds + uploads + downloads the canary `bootstrap` artifact like lex/nova, so the Makefile uses the pre-built binary.
  2. Attempt 2 (b3fd1f7) FAILED at changeset creation: CFN `EarlyValidation::PropertyValidation` — `AWS::Connect::User` requires `SecurityProfileArns` (ARNs), not `SecurityProfileIds`. Fix (37314b7): renamed to `SecurityProfileArns` and build the ARN from instance id + `AgentSecurityProfileId` via `!Sub`. Confirmed `AgentSecurityProfileId` resolves (Agent profile) and the password is passed (masked) — the User will be created.
  3. Added a **cfn-lint fast-fail gate** to deploy.yml's build job (E-level only; warnings don't block) so template property errors surface in seconds instead of after a multi-minute SAM build. (cfn-lint v1.42 catches AWS::Connect::User-style schema errors locally too — now part of the dev loop.)
- A-02 [x] (38 KB metadata sidecars) landed in the 37314b7 push.

### M1 Working Log

---
**[2026-06-12] Claude(Opus orchestrator) — B-01 session store [x]**
- (Sonnet subagent) `internal/session/store.go` + `store_test.go` — `Store` over `SessionTable` mirroring persona.Loader's mockable `dynamoAPI` pattern. `Load` (miss → fresh Session, no error), `Save` (TTL = now+24h per SEC-6, sets LastActivity, **conditional write on last_activity → `ErrConcurrentUpdate`** on ConditionalCheckFailed). Typed accessors (GetInt/SetInt/GetStringSlice/AppendAttemptedStep + named methods) over Session.Attributes. **Key reconciliation:** re-exports triage's 9 `Attr*` consts as session `Key*` (identical strings, asserted by a test); adds session-only keys (attempted_steps, last_response, pace_rate, low_asr_count, no_match_count) per design-doc §8. Did NOT edit triage/types.go. **B-02 must use `session.Key*` / the named methods.** 88.2% coverage; build/vet/test green. Inert until B-02 wires it in.

### M2 Working Log
_(empty)_

### M3 Working Log
_(empty)_

### M4 Working Log
_(empty)_

---

## 5. Open Questions / Decisions

Autonomy rule: if a question blocks a Must task and no answer exists, the workstream lead makes the **most reversible** choice, records it here as PROVISIONAL with rationale, and proceeds; irreversible choices stay `[!]`. The four marked ⚠ are surfaced to the user before execution (they carry real cost/scope weight).

| # | Question | Blocks | Decision | Status |
|---|---|---|---|---|
| Q-A | **Vector store for RAG.** OpenSearch Serverless has a ~$175/mo idle floor. | A-04 → all KB infra | **S3 Vectors** (Bedrock-native, pennies); **fall back to Pinecone free tier** (API key in SSM SecureString) if S3 Vectors isn't GA in us-east-1 at build time. | ✅ decided 2026-06-12 |
| Q-B | **Record calls?** | WS-D-01, REL-6 | **YES — record + transcribe.** Enable Connect call recording + **Contact Lens** to an encrypted (SSE-KMS) S3 bucket with retention; play a recording/transcription consent disclosure as the first block; this also powers QA scorecards (REL-6). Accept the ~$0.015/analyzed-min Contact Lens cost. | ✅ decided 2026-06-12 |
| Q-C | **Staffed human agents, or bot-only + voicemail/callback?** | OPS-1/2, M0 DoD | **Bot-only + voicemail/callback** for v1. Provision the queue + 1 test user, but after-hours/voicemail/callback is the default escalation path. | ✅ decided 2026-06-12 |
| Q-D | **Path B (Nova Sonic): complete or fence?** | REL-4 → WS-H, M1/M2 | **COMPLETE Nova Sonic now** as a v1 goal. Path B is no longer fenced; see new **WS-H** (bidirectional streaming via KVS media + `InvokeModelWithBidirectionalStream`). REL-4's "fence" task (WS-G-09) is repurposed to a safety gate: Path B stays behind `DeployNovaSonicPath` and cannot serve live traffic until WS-H's acceptance tests pass. | ✅ decided 2026-06-12 |
| Q-E | Spanish at launch (AX-4)? | E-4.* | **English-only at launch.** WS-E E-4.* (Spanish locale/personas/KB/IVR) deferred to M4/Later as a config+content add-on (no Go changes needed to add a language later). | ✅ decided 2026-06-12 |
| Q-F | CRM/ticketing target for OPS-8/DM-6? | WS-C-13, DM-6 | **Own data, no CRM** for v1. Repeat-caller context uses the bot's own disposition/session data in DynamoDB. | ✅ decided 2026-06-12 |
| Q-G | Dev/staging environment? | REL-5, WS-G-11 | **Production-only — there is no dev/staging instance.** WS-G-11's ephemeral dev-stack task is dropped; the DR runbook portion is kept. Changes prove out in prod behind the M0 safety net (tests + alarms + gates). Commit directly to `main`. | ✅ decided 2026-06-12 |
| Q-H | Contact Lens coverage (given Q-B recording)? | WS-D-01c, REL-6 | **Analyze 100% of calls** (full QA/analytics fidelity). Cost scales with volume; sampling remains a future lever if needed. | ✅ decided 2026-06-12 |

**All planning questions are resolved — no outstanding questions remain.** The plan is execution-ready.

---

## 6. Risk Register

| # | Risk | L/I | Mitigation |
|---|---|---|---|
| R1 | Prod-only deploys: a bad push breaks live calls | H/H | M0 first (tests+alarms before features); small PRs; WS-G-09 fence; instant-revert convention in the log |
| R2 | Agent verifies against the **wrong AWS account** via local CLI | M/H | Convention §0.4-2 in every task; verification only via pipeline logs/CW |
| R3 | Cloud cost creep (recording/Contact Lens + Nova Sonic Fargate + Bedrock tokens) | M/H | WS-F-07 budget + anomaly alarms land early; S3 Vectors over OpenSearch; WS-H-01 evaluates scale-to-zero; Contact Lens sampling + REL-7 token caps as levers |
| R7 | **Nova Sonic audio-return bridge (WS-H-01) proves infeasible/too-laggy** — getting synthesized audio back into a live Connect call is the least-mature piece | M/H | WS-H-01 is a go/no-go spike *before* WS-H-02+ build; Path B stays behind the WS-G-09 gate; verified rollback-to-Path-A on stream failure; if the spike fails, fall back to the original "fence" posture and report to the user |
| R4 | Escalation ships before a staffed queue exists (Q-C) | M/H | OPS-3 after-hours/voicemail is the default path until Q-C resolved; never promise a human we don't have |
| R5 | Concurrent agents make conflicting edits to shared files (`template.yaml`, contact flows) | M/M | One WS owns each file (named in its first log entry touching it); others request via `[!]` + log note; WS-C-05 single-sources flows first |
| R6 | PII leaks via logs/transcripts before SEC-2 lands | M/H | WS-D-02 is M0, ordered before any new logging; WS-D reviews every M0/M1 PR for log statements |

---

## 7. Cost Summary

Updated for the Q-A…Q-D decisions. This is no longer a sub-$10/mo build — the recording (Q-B) and Nova Sonic (Q-D) choices add real cost; both were explicitly chosen.

**Fixed/idle monthly:** alarms ~$1 · canary custom metrics ~$0.60 · Logs data-protection scan pennies · 2 DID numbers ~$2 · recording-bucket KMS CMK ~$1 · **ECS Fargate streaming service for Nova Sonic ~$10–15** (target near-$0 if WS-H-01 proves scale-to-zero) → **≈ $15–20/mo idle**.
**Usage-based:** Bedrock (Sonnet supervisor + Haiku) · **Nova Sonic per-audio-minute** · **Contact Lens ~$0.015/analyzed-min** (Q-B) · Connect voice minutes · KVS ingestion · Lambda.
**Vector store (Q-A):** S3 Vectors = pennies; Pinecone free tier = $0 (fallback). **OpenSearch Serverless (~$175/mo) avoided.**
**Still avoided:** AWS Secrets Manager ($0.40+/secret/mo), toll-free numbers, standing staging Connect instance, DynamoDB CMK (recording bucket gets a CMK; DynamoDB stays on AWS-owned keys).
**Biggest cost levers if budget tightens later:** Nova Sonic Fargate (scale-to-zero), Contact Lens (sample a % of calls rather than 100%), Bedrock output-token caps (REL-7).

---

*Status legend: `[ ]` todo · `[~]` in progress · `[x]` done · `[!]` blocked. Update markers and append Working Log notes in place as work proceeds.*
