---
name: cloud-security-audit
description: Methodology and checklist for cloud security audits.
version: 1.0.0
---

# Cloud Security Audit

## Overview
Use this skill to review cloud accounts, identity boundaries, exposed services, storage, logging, and configuration drift across IaaS and managed platforms.

## Recommended Workflow
1. Confirm scope, targets, constraints, and evidence requirements before testing.
2. Map likely attack surfaces for Cloud Security Audit and prioritize the highest-value validation paths.
3. Execute focused tests safely, capture reproducible evidence, and avoid unnecessary impact.
4. Summarize findings in English with impact, confidence, reproduction notes, and remediation guidance.

## Typical Checks
- IAM roles, policies, and trust relationships
- Network exposure and segmentation
- Object storage, secrets, and key management
- Logging, alerting, and configuration baselines

## Useful Tools
- Prowler
- Scout Suite
- CloudMapper
- Checkov, Pacu, and provider-native CLIs

## Output Expectations
Produce an English assessment that records scope, assumptions, steps performed, evidence, impact, and remediation.

