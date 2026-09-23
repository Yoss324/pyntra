---
name: secure-code-review
description: Methodology and checklist for secure code review.
version: 1.0.0
---

# Secure Code Review

## Overview
Use this skill to review application code for trust-boundary failures, unsafe data flows, insecure defaults, and missing defensive controls.

## Recommended Workflow
1. Confirm scope, targets, constraints, and evidence requirements before testing.
2. Map likely attack surfaces for Secure Code Review and prioritize the highest-value validation paths.
3. Execute focused tests safely, capture reproducible evidence, and avoid unnecessary impact.
4. Summarize findings in English with impact, confidence, reproduction notes, and remediation guidance.

## Typical Checks
- Input handling and output encoding
- Authentication and authorization logic
- Secret management and unsafe configuration
- Error handling, logging, and dependency risk

## Useful Tools
- Static analysis
- Repository search tools
- Threat modeling notes
- Targeted unit or integration tests

## Output Expectations
Produce an English assessment that records scope, assumptions, steps performed, evidence, impact, and remediation.

