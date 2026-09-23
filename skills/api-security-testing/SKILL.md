---
name: api-security-testing
description: Methodology and checklist for API security testing.
version: 1.0.0
---

# API Security Testing

## Overview
Use this skill when reviewing REST, GraphQL, RPC, or internal service APIs for authentication, authorization, input handling, business logic flaws, and abuse paths.

## Recommended Workflow
1. Confirm scope, targets, constraints, and evidence requirements before testing.
2. Map likely attack surfaces for API Security Testing and prioritize the highest-value validation paths.
3. Execute focused tests safely, capture reproducible evidence, and avoid unnecessary impact.
4. Summarize findings in English with impact, confidence, reproduction notes, and remediation guidance.

## Typical Checks
- Authentication and token handling
- Authorization boundaries between users, roles, and tenants
- Input validation and injection paths
- Rate limiting, error handling, and data exposure

## Useful Tools
- Burp Suite
- Postman or Insomnia
- curl/httpie
- jwt_tool, Arjun, and GraphQL-specific tooling when relevant

## Output Expectations
Produce an English assessment that records scope, assumptions, steps performed, evidence, impact, and remediation.

