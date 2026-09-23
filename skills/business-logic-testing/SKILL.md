---
name: business-logic-testing
description: Methodology and checklist for business logic testing.
version: 1.0.0
---

# Business Logic Testing

## Overview
Use this skill to test workflow abuse, sequencing flaws, state transition weaknesses, and economic or process-level defects that survive standard input validation.

## Recommended Workflow
1. Confirm scope, targets, constraints, and evidence requirements before testing.
2. Map likely attack surfaces for Business Logic Testing and prioritize the highest-value validation paths.
3. Execute focused tests safely, capture reproducible evidence, and avoid unnecessary impact.
4. Summarize findings in English with impact, confidence, reproduction notes, and remediation guidance.

## Typical Checks
- Workflow bypass and missing prerequisites
- State transition abuse and race conditions
- Pricing, discount, or quota manipulation
- Privilege or feature abuse hidden behind valid requests

## Useful Tools
- Burp Suite
- Custom scripts for replay and race testing
- Browser developer tools
- Application logs and business process documentation

## Output Expectations
Produce an English assessment that records scope, assumptions, steps performed, evidence, impact, and remediation.

