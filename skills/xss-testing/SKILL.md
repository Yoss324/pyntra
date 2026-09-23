---
name: xss-testing
description: Methodology and checklist for cross-site scripting testing.
version: 1.0.0
---

# XSS Testing

## Overview
Use this skill to review reflected, stored, and DOM-based XSS across HTML, attribute, JavaScript, URL, and templating contexts.

## Recommended Workflow
1. Confirm scope, targets, constraints, and evidence requirements before testing.
2. Map likely attack surfaces for XSS Testing and prioritize the highest-value validation paths.
3. Execute focused tests safely, capture reproducible evidence, and avoid unnecessary impact.
4. Summarize findings in English with impact, confidence, reproduction notes, and remediation guidance.

## Typical Checks
- Context-aware output encoding failures
- Stored and reflected injection paths
- DOM sinks and client-side templating issues
- CSP, sanitization, and session impact

## Useful Tools
- Burp Suite
- Browser developer tools
- DOM inspection helpers
- Context-specific payload sets

## Output Expectations
Produce an English assessment that records scope, assumptions, steps performed, evidence, impact, and remediation.

