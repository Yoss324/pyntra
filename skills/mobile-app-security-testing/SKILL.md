---
name: mobile-app-security-testing
description: Methodology and checklist for mobile application security testing.
version: 1.0.0
---

# Mobile App Security Testing

## Overview
Use this skill to review mobile storage, local trust decisions, API usage, client-side protections, and release artifacts for Android and iOS apps.

## Recommended Workflow
1. Confirm scope, targets, constraints, and evidence requirements before testing.
2. Map likely attack surfaces for Mobile App Security Testing and prioritize the highest-value validation paths.
3. Execute focused tests safely, capture reproducible evidence, and avoid unnecessary impact.
4. Summarize findings in English with impact, confidence, reproduction notes, and remediation guidance.

## Typical Checks
- Sensitive data in local storage and logs
- Certificate pinning and transport security behavior
- Deep links, intents, and exported components
- Hardcoded secrets and backend trust assumptions

## Useful Tools
- Burp Suite
- MobSF or similar analysis tools
- adb or platform-native device tooling
- Static code review and binary inspection tools

## Output Expectations
Produce an English assessment that records scope, assumptions, steps performed, evidence, impact, and remediation.

