---
name: deserialization-testing
description: Methodology and checklist for insecure deserialization testing.
version: 1.0.0
---

# Deserialization Testing

## Overview
Use this skill to review data formats, parsers, and object reconstruction paths that may allow unexpected object graphs, gadget execution, or trust-boundary bypasses.

## Recommended Workflow
1. Confirm scope, targets, constraints, and evidence requirements before testing.
2. Map likely attack surfaces for Deserialization Testing and prioritize the highest-value validation paths.
3. Execute focused tests safely, capture reproducible evidence, and avoid unnecessary impact.
4. Summarize findings in English with impact, confidence, reproduction notes, and remediation guidance.

## Typical Checks
- Unsigned or weakly protected serialized objects
- Dangerous type restoration and gadget availability
- Cross-language or format-specific parser quirks
- Impact reduction through integrity checks and safer formats

## Useful Tools
- Burp Suite
- Language-specific serializer utilities
- Static code review
- Safe proof-of-concept payload builders

## Output Expectations
Produce an English assessment that records scope, assumptions, steps performed, evidence, impact, and remediation.

