---
name: command-injection-testing
description: Methodology and checklist for command injection testing.
version: 1.0.0
---

# Command Injection Testing

## Overview
Use this skill to review features that build shell commands, spawn processes, or pass user-controlled data into operating-system utilities.

## Recommended Workflow
1. Confirm scope, targets, constraints, and evidence requirements before testing.
2. Map likely attack surfaces for Command Injection Testing and prioritize the highest-value validation paths.
3. Execute focused tests safely, capture reproducible evidence, and avoid unnecessary impact.
4. Summarize findings in English with impact, confidence, reproduction notes, and remediation guidance.

## Typical Checks
- Unsafe concatenation and shell metacharacter handling
- Blind injection channels and out-of-band confirmation paths
- Environment-variable and path-based abuse
- Privilege context and post-execution impact

## Useful Tools
- Burp Suite
- curl/httpie
- dnslog or collaborator-style OOB services where permitted
- Targeted payload lists and safe validation scripts

## Output Expectations
Produce an English assessment that records scope, assumptions, steps performed, evidence, impact, and remediation.

