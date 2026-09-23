---
name: idor-testing
description: Methodology and checklist for insecure direct object reference testing.
version: 1.0.0
---

# IDOR Testing

## Overview
Use this skill to review object references, identifier formats, and authorization checks around record access and modification.

## Recommended Workflow
1. Confirm scope, targets, constraints, and evidence requirements before testing.
2. Map likely attack surfaces for IDOR Testing and prioritize the highest-value validation paths.
3. Execute focused tests safely, capture reproducible evidence, and avoid unnecessary impact.
4. Summarize findings in English with impact, confidence, reproduction notes, and remediation guidance.

## Typical Checks
- Horizontal access between users or tenants
- Vertical access to privileged objects
- Predictable identifiers and enumeration paths
- Bulk operations that skip per-object authorization

## Useful Tools
- Burp Suite
- Custom identifier mutation scripts
- Browser developer tools
- Application audit logs where available

## Output Expectations
Produce an English assessment that records scope, assumptions, steps performed, evidence, impact, and remediation.

