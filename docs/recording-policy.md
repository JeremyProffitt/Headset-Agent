# Call Recording & AI Disclosure Policy

_Headset Support Agent — Amazon Connect voice line_

## What is recorded

When you call the Headset Support line, the following may be captured:

- **Audio recording** of the call (both the caller's and the AI assistant's
  side of the conversation).
- **Transcription** of the call (speech-to-text), produced for quality, training,
  and troubleshooting.
- **Contact metadata** held by Amazon Connect: the contact identifier, call
  timestamps, the persona selected, the issue type, and whether the call was
  escalated. The caller's phone number (ANI) is handled as PII — it is masked in
  application logs and, where used for repeat-caller context, is stored only as a
  salted SHA-256 hash, never in raw form.

The call is also handled by an **automated AI assistant** rather than a human for
the initial troubleshooting portion of the call.

> Note: the audio/transcript **storage** infrastructure (encrypted S3 bucket,
> Connect instance storage config, and Contact Lens analytics) is provisioned by a
> separate task (WS-D-01b/01c). This document covers the consent disclosure and the
> retention/consent stance.

## Retention

- **Recordings and transcripts: 90 days**, after which they are deleted.
- CloudWatch application logs follow the same **90-day** retention in production
  and have a data-protection policy that masks phone numbers, names, and addresses.

## Consent stance (two-party consent)

The service operates on a **disclosure-then-implied-consent** model:

- The **first audible message** on every call states clearly that the call is
  recorded and transcribed for quality and training and that it is handled by an
  automated AI assistant, and instructs the caller to hang up if they do not
  consent.
- A caller who **remains on the line after the disclosure is treated as having
  consented** to recording, transcription, and AI handling.
- This satisfies all-party / two-party-consent jurisdictions (e.g. California,
  Florida, Pennsylvania) by giving every party clear notice and a meaningful
  opportunity to decline before any substantive recording occurs.

The exact disclosure wording lives in the `consent` action — the `StartAction` —
of the `LexContactFlow` resource in `infrastructure/template.yaml`. Any change to
the recorded-disclosure language must be made there (the single source of truth)
and reflected here.

## Who to contact

Questions, access requests, or deletion requests regarding recordings or
transcripts: **proffitt.jeremy@gmail.com**.
