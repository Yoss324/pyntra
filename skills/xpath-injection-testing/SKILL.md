---
name: xpath-injection-testing
description: Methodology and checklist for XPath injection testing.
version: 1.0.0
---

# XPath Injection Testing

## Overview
Use this skill to review XPath-backed authentication, search, and filtering logic where untrusted input is inserted into XML query expressions.

## Recommended Workflow
1. Confirm scope, targets, constraints, and evidence requirements before testing.
2. Map likely attack surfaces for XPath Injection Testing and prioritize the highest-value validation paths.
3. Execute focused tests safely, capture reproducible evidence, and avoid unnecessary impact.
4. Summarize findings in English with impact, confidence, reproduction notes, and remediation guidance.

## Typical Checks
- Authentication bypass through crafted XPath predicates
- Blind extraction through boolean or timing differences
- Input escaping and query construction flaws
- Safer parsing and query API recommendations

## Useful Tools
- Burp Suite
- Custom XPath payloads
- Application logs
- Controlled XML test fixtures

## Output Expectations
Produce an English assessment that records scope, assumptions, steps performed, evidence, impact, and remediation.

