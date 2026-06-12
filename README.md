# Headset Support Agent

Voice-based headset troubleshooting agent with programmable personas.

## Project Status

Starting fresh - architecture to be determined.

## Personas

The system supports three distinct agent personas:

| Persona | Origin | Personality |
|---------|--------|-------------|
| **Tangerine** | Dublin, Ireland | Young, upbeat, enthusiastic |
| **Joseph** | Columbus, Ohio | Calm, patient, methodical |
| **Jennifer** | Rural Nebraska | Fast-talking, folksy, energetic |

## Knowledge Base

Troubleshooting documentation is organized in the `knowledge-base/` directory:

- `usb/` - USB headset troubleshooting
- `bluetooth/` - Bluetooth pairing and connectivity
- `windows/` - Windows audio configuration
- `genesys-cloud/` - Genesys Cloud Desktop setup
- `common/` - FAQs and general information

## Test Chat Front-End

A simple password-gated web chat for testing the bot is published to the website
bucket by the deploy pipeline (`publish-website` job):

- **URL:** `http://headset-chat-<aws-account-id>-prod.s3-website-us-east-1.amazonaws.com`
  (the exact URL is printed in the deploy run's **Publish Test Chat Front-End** job
  and is the stack's `WebsiteUrl` output).
- **Test password:** `testing-me`

> ⚠️ This is a **test-only soft gate** — the password is enforced client-side and
> ships in the page source. It is not real authentication. Real API auth + a
> CloudFront/OAC lockdown of the bucket are tracked as SEC-5 (WS-D-07 / WS-D-08).

## Documentation

- [Persona & Troubleshooting Guide](docs/persona-troubleshooting-guide.md) - Agent personas and troubleshooting methodology
