# Headset Support Agent Project

## Project Overview
Voice-based headset troubleshooting agent using AWS multi-agent architecture with programmable personas.

## Key Documentation
- docs/headset-agent-implementation-guide.md - Architecture and implementation
- docs/persona-troubleshooting-guide.md - Personas and troubleshooting flows
- docs/deployment-guide.md - GitHub Actions CI/CD with Claude Code autonomy
- docs/variables.md - GitHub secrets and variables
- docs/regions.md - AWS region requirements

## Critical Requirements
1. Deploy via GitHub Actions ONLY - No local deployments
2. Claude Code has full autonomy for builds and fixes
3. Primary region: us-east-1 (required for Amazon Connect)
4. Three personas: Tangerine (Irish), Joseph (Ohio), Jennifer (Farm)
