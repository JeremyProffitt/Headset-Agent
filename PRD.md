# Product Requirements Document — Headset Support Agent

**Version:** 1.0 (discovery / baseline)
**Status:** Draft for review
**Date:** 2026-06-12
**Owner:** Product (Jeremy Proffitt)
**Authoring method:** Synthesized from a five-persona review (Product, CX Operations, Conversational-AI/Voice, Security & Reliability Architecture, Domain & Accessibility) conducted against the actual codebase at `C:\dev\Headset-Agent`.

---

## 1. Overview

### 1.1 Product summary
The **Headset Support Agent** is a voice-and-chat AI that answers inbound customer contacts (phone via Amazon Connect, and web chat) and walks callers through resolving headset problems — USB no-audio, Bluetooth pairing, Windows sound configuration, and Genesys Cloud softphone setup — the way a tier-1 contact-center agent would. Its differentiator is a set of **programmable, regionally-flavored personas** (Tangerine/Irish, Joseph/Ohio, Jennifer/Nebraska), each with its own voice, prosody, and scripted phrasing.

### 1.2 Users
- **End callers** — contact-center agents and office workers whose headset has stopped working and who need a fast, guided fix or a clean handoff to a human.
- **The support organization** — operations, QA, and support engineers who want call deflection (containment), consistent quality, escalation with context, and reporting.

### 1.3 Architecture context (as-built)
The system deliberately ships **two parallel voice paths** for A/B comparison, both feeding a shared Amazon Bedrock backend:
- **Path A — Amazon Lex V2 → Bedrock Agent (Claude 3.5 Sonnet).** ~80% functional today.
- **Path B — Amazon Bedrock Nova Sonic streaming.** Currently a **stub** (no real bidirectional audio).

Deployment is **GitHub Actions only** into a single prod stack (`agent-headset-prod`, us-east-1). Local AWS CLI targets a different account and must not be used for deploys (see `CLAUDE.md`).

> **Convergence recommendation (from the voice-engineering review):** Treat **Path A as the production path** and **Path B (Nova Sonic) as a contained R&D spike**. The highest-leverage work (knowledge grounding, streaming TTS, barge-in, real escalation) is independent of which front-end wins. Revisit Nova Sonic once it is GA in-region and Path-A conversation quality is proven.

---

## 2. Current State — Verified Feature Inventory

Legend: ✅ Real & working · 🟡 Partial · 🔶 Stubbed/dead · ❌ Planned but absent

| Feature | State | Evidence / note |
|---|---|---|
| Three personas with full voice config | ✅ | `personas/*.json` — Polly + Nova Sonic voices, prosody, system prompts, phrase banks |
| Persona loader with DynamoDB + fallback | ✅ | `internal/persona/persona.go` (`DefaultPersona()` on miss) |
| Lex (Path A) fulfillment Lambda | ✅ | `cmd/lex-lambda/main.go` — parses Lex V2 events, invokes Bedrock agent, builds SSML; also serves an HTTP `/chat` path |
| SSML response builder (per-persona prosody) | ✅ | `internal/handlers/lex.go` `BuildSSML()`, HTML-escapes input |
| Bedrock agent invocation (streaming + cleanup) | ✅ | `internal/agents/bedrock.go` — 25s timeout, persona via session attributes |
| Rule-based escalation **detection** | 🟡 | `internal/handlers/escalation.go` — keyword + frustration/step counters. *Counters are read but never incremented*, so cumulative thresholds rarely fire |
| Knowledge-base content (5 docs) | 🟡 | `knowledge-base/**` — good content, **not wired to any retrieval system** |
| Dual-path infrastructure | ✅ | `infrastructure/template.yaml` — S3, DynamoDB (Persona + TTL Session), both Lambdas (arm64), Lex bot/alias, Connect instance + 2 contact flows, chat HTTP API, website bucket, SSM params |
| CI/CD pipeline | ✅ | `deploy.yml`, `pr-validation.yml`, `destroy.yml` |
| Bedrock multi-agent collaboration | 🔶 | `scripts/create-agents.py` creates 4 agents but never calls `associate_agent_collaborator`, attaches action groups, or associates a KB — sub-agents are orphaned |
| Bedrock Knowledge Base / RAG | ❌ | No `AWS::Bedrock::KnowledgeBase`, no vector store; deploy only `s3 sync`s the docs. Agent answers from parametric memory only |
| Nova Sonic (Path B) bidirectional voice | 🔶 | `cmd/nova-sonic-lambda/main.go` — explicit `// TODO: Implement actual Nova Sonic bidirectional streaming` |
| Human escalation **routing** | 🔶 | Contact flows have a transfer block with **empty params and no queue**. No `AWS::Connect::Queue`/`RoutingProfile`/`HoursOfOperation`/`User` exist anywhere |
| Conversation memory / session store | 🔶 | TTL `SessionTable` provisioned but **no code reads/writes it**; state lives only in Lex attributes within a single call |
| Persona-selection IVR | ❌ | No DTMF/voice menu; `persona_id` is never set → everything defaults to `tangerine` |
| Lex slots (brand/connection/issue) | ❌ | `TroubleshootIntent` has utterances but **no slots** |
| Sentiment-driven escalation | ❌ | `DetectSentiment: false` on the Lex alias; frustration is keyword-only |
| Call recording / transcription | ❌ | No Connect `InstanceStorageConfig`, no Contact Lens, no Kinesis |
| Automated tests | ❌ | `tests/` empty; zero `*_test.go`. Pipeline `go test` passes vacuously; integration tests run with `continue-on-error` |
| Observability (alarms/dashboards/tracing) | ❌ | None built (TODO Phase 12 unimplemented) |

### 2.1 The five "honest truths" that anchor scope
1. **Knowledge is disconnected.** The product's core value — guided, brand-aware troubleshooting — is not reachable by the agent (KB exists as files only).
2. **Escalation is a dead end.** The bot promises a human transfer that routes to an empty, queue-less block. No human destination exists.
3. **Path B is a stub.** The headline "dual-path A/B" cannot actually run.
4. **Memory is provisioned but unused**, which undermines the frustration/step-count escalation logic.
5. **No tests, no observability, prod-only deploys** — changes test in production with no safety net, and outages are invisible.

---

## 3. Goals & Non-Goals

### 3.1 Goals (v1)
- Turn the bot into a **genuinely helpful tier-1 troubleshooter** for USB/Bluetooth/Windows/softphone headset issues, grounded in a real knowledge base.
- Provide a **reliable, context-rich path to a human** when the bot can't resolve the issue or the caller asks.
- Be **operable and safe in production**: recording consent, PII handling, tests, monitoring, and a real escalation queue.
- Be **inclusive**: callers who can't hear well, don't speak English natively, or aren't tech-savvy can still get help.

### 3.2 Non-Goals (v1)
- Full production Nova Sonic bidirectional voice (R&D spike only).
- Payment processing / any PCI-in-scope flows.
- Outbound calling / proactive campaigns.
- Native mobile app.

---

## 4. User Journeys

| # | Journey | Works today | Primary gaps |
|---|---|---|---|
| J1 | USB headset, no audio → resolution | Speech capture, persona voice, single-turn agent answer | No KB grounding; no step-state across turns |
| J2 | Bluetooth pairing failure (brand-specific) | Same as J1 | No brand slot; brand-specific steps not retrievable |
| J3 | Caller explicitly escalates to a human | Detection + spoken handoff promise | **Transfer has no destination** — call ends |
| J4 | Frustrated caller (implicit escalation) | Keyword detection | Counters never persist/increment; sentiment off |
| J5 | Web-chat (text) user | Text round-trip to agent | No published UI, no auth/rate-limit, same KB/memory gaps |
| J6 | Hearing-impaired / non-English caller | — | **No fallback channel, no multilingual, no pace control** |

---

## 5. Requirements

Each requirement carries a MoSCoW priority and rough effort (S/M/L). Items marked **(blocker)** must ship before the channel can go live.

### 5.1 Core conversational capability

| ID | Requirement | Priority | Effort |
|---|---|---|---|
| CC-1 | **Wire a Bedrock Knowledge Base (RAG).** Create KB from `knowledge-base/**` (Titan embeddings + OpenSearch Serverless), associate to the agent so answers use the real troubleshooting content. **(blocker)** | Must | M |
| CC-2 | **Symptom-first triage tree.** Deterministic top-level fork (input vs output vs connection vs call-quality → connection type → app) that drives the per-doc diagnosis questions, rather than relying on the LLM to improvise. | Must | M |
| CC-3 | **"Did it work?" verification loop + step-state.** Track attempted steps in the session; after each suggested fix, confirm; on failure advance and increment `failed_steps`. | Must | M |
| CC-4 | **Conversation memory.** Actually read/write the `SessionTable`: turn state, attempted steps, frustration/step counters, recent history. **(blocker for CC-3/escalation)** | Must | M |
| CC-5 | **Lex slots.** Capture `connection_type`, `headset_brand`, `issue_type` with confirmation + DTMF/spelling fallback to target retrieval. | Should | M |
| CC-6 | **Hallucination guardrails.** Bedrock Guardrails (denied topics, "never fabricate steps," PII filter) + grounding/relevance check. | Should | S |
| CC-7 | **Sub-agent collaboration — wire or collapse.** Either configure supervisor→Diagnostic/Platform/Escalation collaboration, or collapse to one tool-using agent. Do not ship orphaned sub-agents. | Should | M |

### 5.2 Voice experience

| ID | Requirement | Priority | Effort |
|---|---|---|---|
| VX-1 | **Streaming / partial TTS.** Stream agent tokens to Polly/Connect incrementally instead of buffering the whole response (removes multi-second dead air). | Must | M |
| VX-2 | **Barge-in.** Let callers interrupt prompts; configure Lex/Connect interruptibility. | Must | S |
| VX-3 | **No-input / no-match escalating reprompts** → rephrase → DTMF fallback → human. | Should | S |
| VX-4 | **Persona-selection IVR** (DTMF or speech) that actually sets `persona_id`. | Should | S |
| VX-5 | **Caller-adjustable pace + "repeat / slow down / speak up" commands** that re-emit the last turn at a reduced SSML rate and persist the preference. | Must | S |
| VX-6 | **Low-confidence ASR / accent fallback.** On repeated low transcription confidence, offer keypad options and a faster human handoff. | Should | M |

### 5.3 Contact-center operations

| ID | Requirement | Priority | Effort |
|---|---|---|---|
| OPS-1 | **Real escalation queue + agents.** Provision `AWS::Connect::Queue`, `RoutingProfile`, `HoursOfOperation`, ≥1 `User`; wire `TransferContactToQueue`. **(blocker)** | Must | M |
| OPS-2 | **Warm transfer with context.** Pass detected issue, transcript summary, steps tried, persona, and escalation reason as contact attributes; whisper-brief the agent before connect. **(blocker)** | Must | M |
| OPS-3 | **Business hours + after-hours flow** (closed-hours branch → callback/voicemail/message). **(blocker)** | Must | S |
| OPS-4 | **Queued callback** ("keep your place, we'll call you back") when wait exceeds a threshold. | Should | M |
| OPS-5 | **Voicemail capture** when unstaffed / after hours, routed to a ticket/email. | Should | S |
| OPS-6 | **CSAT post-contact survey** (DTMF 1–5 or SMS) for bot-only vs escalated contacts. | Should | S |
| OPS-7 | **Wrap-up / disposition codes** (contained / escalated / abandoned / resolved) for every contact. | Should | S |
| OPS-8 | **Repeat-caller recognition** via ANI lookup against CRM/ticketing; personalize and skip repeated questions. | Should | M |
| OPS-9 | **Skills-based routing** mapping detected issue (hardware/platform/Genesys) to agent skill queues. | Could | M |
| OPS-10 | **Supervisor monitor / whisper / barge.** | Could | S |

### 5.4 Domain coverage (see `troubleshooting.md` for the detailed USB-on-Windows tree)

| ID | Requirement | Priority | Effort |
|---|---|---|---|
| DM-1 | **Microphone troubleshooting** as a first-class flow (not working / too quiet / muffled / mute-LED / levels & boost / app-permission vs privacy). The #1 contact-center headset complaint, near-zero coverage today. | Must | M |
| DM-2 | **DECT / USB-dongle wireless flow** (Jabra Link, Poly BT700/Savi, Yealink DECT) — pairing, base reset, range, re-sync. | Should | M |
| DM-3 | **Teams / Zoom / Webex device-config KB** (not Genesys only) — per-app device selection, "certified for Teams" behavior, in-app test. | Should | M |
| DM-4 | **Firmware / driver / software guidance per brand** (Jabra Direct, Poly Lens/Hub, Logi Tune) — update + clean-reinstall + known-bad-firmware notes. | Should | M |
| DM-5 | **Audio-quality flows** (distortion, crackle, one-sided/mono, echo, sidetone, latency). | Should | M |
| DM-6 | **Warranty / RMA capture** — collect model/serial/warranty status; pass structured summary on handoff. | Could | M |
| DM-7 | **macOS / mobile coverage.** | Could | L |
| DM-8 | **Known-defect / advisory lookup** the agent can surface proactively. | Could | M |

### 5.5 Accessibility & inclusion

| ID | Requirement | Priority | Effort |
|---|---|---|---|
| AX-1 | **SMS / web-chat / email-the-steps fallback** for Deaf/HoH callers (an audio-diagnosis bot is structurally hostile to people who can't hear it). **(highest inclusion risk)** | Must | M |
| AX-2 | **Pace control** (shared with VX-5). | Must | S |
| AX-3 | **Plain-language / clear-and-slow persona** — neutral accent, no idioms, literal, repeats key actions — as a safe default for low-tech/elderly/non-native callers. | Should | S |
| AX-4 | **Spanish (then multi-language)** — language-select IVR, translated personas + KB, localized voices. | Should | L |
| AX-5 | **Non-visual / non-auditory cue alternatives** — replace "blue blinking LED" / "listen for the chime" success checks with described alternatives. | Could | S |
| AX-6 | **TTY / TRS (711 relay) compatibility.** | Could | M |

### 5.6 Security, privacy & compliance

| ID | Requirement | Priority | Effort |
|---|---|---|---|
| SEC-1 | **Call-recording consent + recording policy.** Disclosure prompt as the first contact-flow block; decide/document record vs no-record; if recording, store to an encrypted bucket with retention. **(legal blocker if recording)** | Must | S |
| SEC-2 | **PII redaction in logs.** Stop logging raw events/transcripts/caller-ID; structured JSON logging; CloudWatch Logs data-protection masking; consider Lex/Comprehend PII obfuscation. **(blocker)** | Must | M |
| SEC-3 | **GitHub OIDC** role-assumption instead of long-lived `AWS_ACCESS_KEY_ID/SECRET`. | Should | S |
| SEC-4 | **Least-privilege Bedrock IAM** — scope `bedrock:Invoke*` to specific model/inference-profile ARNs (currently `*`). | Should | S |
| SEC-5 | **Lock down `/chat` + website** — auth/API-key on `/chat`, restrict CORS to known origins, front the website with CloudFront + TLS + OAC and re-enable public-access block (bucket is currently world-readable over `http://`). | Should | M |
| SEC-6 | **Data retention & DSAR** — wire up (or remove) the session table; define transcript/log retention; build delete-by-caller; data inventory (GDPR Art. 17 / CCPA). | Should | M |
| SEC-7 | **No-payment guardrail** — the bot must never solicit/capture card data; if ever required, hand to a PCI-compliant DTMF capture. | Must | S |

### 5.7 Reliability, quality & observability

| ID | Requirement | Priority | Effort |
|---|---|---|---|
| REL-1 | **Real automated tests + honest CI gates.** Unit tests for `escalation`, `lex`, `bedrock`, `persona`; remove `continue-on-error`/`|| true`; fail PRs on a real coverage threshold; stop hardcoding ✅ summaries. **(blocker)** | Must | M |
| REL-2 | **Observability stack.** CloudWatch dashboard; alarms (Lambda error% / duration / throttle, Bedrock throttling, Connect abandoned/queue, phone-claim failure, agent-still-PLACEHOLDER); SNS → Slack/PagerDuty; X-Ray tracing on both Lambdas. **(blocker)** | Must | M |
| REL-3 | **DLQ + idempotency + retry config** on both Lambdas; idempotency key per Connect contact turn; idempotent agent/phone provisioning. | Should | M |
| REL-4 | **Complete or fence off Path B** so a stubbed/looping path can't take live calls. **(blocker)** | Must | L |
| REL-5 | **dev/staging environment + DR runbook** (RTO/RPO, restore steps, KMS CMK on DynamoDB) so changes don't test in prod. | Could | L |
| REL-6 | **Quality & governance** — recording + Contact Lens transcripts; QA scorecards (including LLM-hallucination/KB-grounding review); supervisor monitoring. | Should | M |
| REL-7 | **Cost guardrails** — AWS Budgets + anomaly alerts; Lambda reserved concurrency; Bedrock output-token caps + prefer Haiku where Sonnet isn't needed; orphan toll-free-number sweeper. | Should | S |

---

## 6. Success Metrics / KPIs

**North-star**
- **Containment / self-service deflection rate** — % of contacts resolved by the bot without human transfer (requires reliable disposition capture; today escalation fails so this number is meaningless).
- **First-Contact Resolution (FCR)** — requires the "Did that fix it?" confirmation (CC-3) + repeat-contact tracking by ANI.

**Experience & quality**
- CSAT / post-contact survey (bot-only vs escalated); chat thumbs-up rate.
- Escalation rate by reason (`user_requested` / `user_frustrated` / `troubleshooting_exhausted` — already classified in code; just emit as metric dimensions).
- KB retrieval hit rate / low-confidence-answer rate → feeds the knowledge-gap loop.

**Operational**
- Service level / ASA, abandonment rate, transfer success rate, AHT and turns-to-resolution per issue category.
- **Per-path comparison** (containment, CSAT, escalation, AHT for `Lex` vs `NovaSonic`) — the stated reason the dual paths and personas exist.

**Engineering**
- Bedrock agent latency (targets: Lex <3s, Nova Sonic <2s) and timeout rate (25s agent timeout).
- Cost per contained contact (Bedrock Sonnet/Haiku + Connect minutes + Lambda).

---

## 7. Phased Roadmap (proposed)

**Phase 0 — Make it safe & honest (blockers).** SEC-1, SEC-2, REL-1, REL-2, REL-4, OPS-1, OPS-2, OPS-3.
**Phase 1 — Make it actually helpful.** CC-1, CC-2, CC-3, CC-4, VX-1, VX-2, DM-1.
**Phase 2 — Make it inclusive & measurable.** AX-1, AX-2/VX-5, OPS-6, OPS-7, VX-3, VX-4, CC-5, CC-6, knowledge-gap analytics.
**Phase 3 — Broaden & optimize.** DM-2…DM-5, OPS-4/5/8, AX-3/4, SEC-3/4/5/6, REL-3/6/7.
**Later / R&D.** Path B Nova Sonic production, OPS-9/10, DM-6/7/8, AX-5/6, REL-5.

---

## 8. Open Questions / Decisions Needed
1. **Record calls or not?** Determines SEC-1 scope and the entire QA/Contact Lens story.
2. **Path B:** invest to complete Nova Sonic, or formally descope to R&D for v1?
3. **Human agents:** are there staffed agents to escalate to, or is v1 bot-only with voicemail/callback fallback?
4. **CRM/ticketing system** for OPS-8 / DM-6 (which one)?
5. **Languages** required at launch beyond English (Spanish?).
6. **Multi-environment:** reinstate dev/staging, or stay prod-only with stronger gates?

---

*Companion documents: `troubleshooting.md` (detailed USB-on-Windows diagnostic paths), `TODO.md` (original dual-path build plan).*
