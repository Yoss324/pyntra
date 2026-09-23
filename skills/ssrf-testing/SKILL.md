---
name: ssrf-testing
description: Methodology and checklist for server-side request forgery testing.
version: 1.0.0
---

# SSRF Testing

## Overview
Use this skill to review server-side URL fetching, callback features, integration clients, and metadata service exposure.

## Recommended Workflow
1. Confirm scope, targets, constraints, and evidence requirements before testing.
2. Map likely attack surfaces for SSRF Testing and prioritize the highest-value validation paths.
3. Execute focused tests safely, capture reproducible evidence, and avoid unnecessary impact.
4. Summarize findings in English with impact, confidence, reproduction notes, and remediation guidance.

## Typical Checks
- URL parser confusion and redirect handling
- Access to cloud metadata or internal services
- Protocol smuggling and alternate schemes
- Network egress controls and allowlists

## Useful Tools
- Burp Suite
- Collaborator-style services where permitted
- Cloud metadata references
- Custom URL mutation payloads

## Output Expectations
Produce an English assessment that records scope, assumptions, steps performed, evidence, impact, and remediation.

