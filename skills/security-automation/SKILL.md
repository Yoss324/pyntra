---
name: security-automation
description: Methodology and checklist for security automation workflows.
version: 1.0.0
---

# Security Automation

## Overview
Use this skill to design repeatable automation for reconnaissance, validation, reporting, and remediation support without losing auditability.

## Recommended Workflow
1. Confirm scope, targets, constraints, and evidence requirements before testing.
2. Map likely attack surfaces for Security Automation and prioritize the highest-value validation paths.
3. Execute focused tests safely, capture reproducible evidence, and avoid unnecessary impact.
4. Summarize findings in English with impact, confidence, reproduction notes, and remediation guidance.

## Typical Checks
- Clear input and output contracts
- Error handling and retry behavior
- Safe secret handling and environment isolation
- Structured evidence and report generation

## Useful Tools
- Shell and Python helpers
- CI/CD jobs
- Task runners and schedulers
- Structured output formats such as JSON and Markdown

## Output Expectations
Produce an English assessment that records scope, assumptions, steps performed, evidence, impact, and remediation.

