---
name: xxe-testing
description: Methodology and checklist for XML external entity testing.
version: 1.0.0
---

# XXE Testing

## Overview
Use this skill to review XML parsers, entity resolution behavior, and backend integrations that may expose local files or internal network resources.

## Recommended Workflow
1. Confirm scope, targets, constraints, and evidence requirements before testing.
2. Map likely attack surfaces for XXE Testing and prioritize the highest-value validation paths.
3. Execute focused tests safely, capture reproducible evidence, and avoid unnecessary impact.
4. Summarize findings in English with impact, confidence, reproduction notes, and remediation guidance.

## Typical Checks
- External entity resolution and DTD processing
- File disclosure and SSRF-like impact
- Parser configuration and safer defaults
- Evidence capture without unnecessary data exposure

## Useful Tools
- Burp Suite
- Controlled XML payloads
- Parser and framework documentation
- Application logs and safe callback endpoints

## Output Expectations
Produce an English assessment that records scope, assumptions, steps performed, evidence, impact, and remediation.

