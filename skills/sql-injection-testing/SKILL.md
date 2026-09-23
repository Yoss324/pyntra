---
name: sql-injection-testing
description: Methodology and checklist for SQL injection testing.
version: 1.0.0
---

# SQL Injection Testing

## Overview
Use this skill to review query construction, ORM boundaries, database error handling, and response differences for injection opportunities.

## Recommended Workflow
1. Confirm scope, targets, constraints, and evidence requirements before testing.
2. Map likely attack surfaces for SQL Injection Testing and prioritize the highest-value validation paths.
3. Execute focused tests safely, capture reproducible evidence, and avoid unnecessary impact.
4. Summarize findings in English with impact, confidence, reproduction notes, and remediation guidance.

## Typical Checks
- Boolean, error-based, union-based, and time-based paths
- Second-order injection in stored data flows
- Access to sensitive tables or metadata
- Safer query construction and parameterization guidance

## Useful Tools
- sqlmap
- Burp Suite
- Database-aware payload lists
- Application and database logs where available

## Output Expectations
Produce an English assessment that records scope, assumptions, steps performed, evidence, impact, and remediation.

