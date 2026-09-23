---
name: csrf-testing
description: Methodology and checklist for CSRF testing.
version: 1.0.0
---

# CSRF Testing

## Overview
Use this skill to review state-changing requests, anti-CSRF tokens, cookie policies, and browser trust boundaries.

## Recommended Workflow
1. Confirm scope, targets, constraints, and evidence requirements before testing.
2. Map likely attack surfaces for CSRF Testing and prioritize the highest-value validation paths.
3. Execute focused tests safely, capture reproducible evidence, and avoid unnecessary impact.
4. Summarize findings in English with impact, confidence, reproduction notes, and remediation guidance.

## Typical Checks
- Missing or ineffective anti-CSRF tokens
- Cookie SameSite and origin validation gaps
- Cross-origin request handling and CORS interactions
- High-impact state changes reachable from a victim browser

## Useful Tools
- Burp Suite
- HTML proof-of-concept pages
- Browser developer tools
- Application session tracing

## Output Expectations
Produce an English assessment that records scope, assumptions, steps performed, evidence, impact, and remediation.

