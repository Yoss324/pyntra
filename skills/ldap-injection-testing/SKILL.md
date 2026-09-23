---
name: ldap-injection-testing
description: Methodology and checklist for LDAP injection testing.
version: 1.0.0
---

# LDAP Injection Testing

## Overview
Use this skill to review LDAP-backed authentication, directory search, and filter construction paths for escaping and logic flaws.

## Recommended Workflow
1. Confirm scope, targets, constraints, and evidence requirements before testing.
2. Map likely attack surfaces for LDAP Injection Testing and prioritize the highest-value validation paths.
3. Execute focused tests safely, capture reproducible evidence, and avoid unnecessary impact.
4. Summarize findings in English with impact, confidence, reproduction notes, and remediation guidance.

## Typical Checks
- Unsafe filter construction
- Authentication bypass through crafted directory input
- Blind enumeration via response differences
- Least-privilege directory access and query scoping

## Useful Tools
- Burp Suite
- Custom LDAP payload sets
- Application logs
- Directory-aware test accounts where permitted

## Output Expectations
Produce an English assessment that records scope, assumptions, steps performed, evidence, impact, and remediation.

