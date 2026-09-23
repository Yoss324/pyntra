---
name: network-penetration-testing
description: Methodology and checklist for network penetration testing.
version: 1.0.0
---

# Network Penetration Testing

## Overview
Use this skill to map reachable hosts, enumerate exposed services, validate segmentation, and assess post-enumeration attack paths within an authorized environment.

## Recommended Workflow
1. Confirm scope, targets, constraints, and evidence requirements before testing.
2. Map likely attack surfaces for Network Penetration Testing and prioritize the highest-value validation paths.
3. Execute focused tests safely, capture reproducible evidence, and avoid unnecessary impact.
4. Summarize findings in English with impact, confidence, reproduction notes, and remediation guidance.

## Typical Checks
- Host discovery and service fingerprinting
- Weak protocols and exposed administrative services
- Segmentation and trust boundary gaps
- Credential reuse and lateral follow-up opportunities

## Useful Tools
- nmap
- masscan or rustscan
- netexec, smbmap, and related protocol tooling
- Packet captures and host inventory data

## Output Expectations
Produce an English assessment that records scope, assumptions, steps performed, evidence, impact, and remediation.

