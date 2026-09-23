---
name: container-security-testing
description: Methodology and checklist for container security testing.
version: 1.0.0
---

# Container Security Testing

## Overview
Use this skill to assess container images, runtime isolation, Kubernetes resources, registry hygiene, and supporting cloud controls.

## Recommended Workflow
1. Confirm scope, targets, constraints, and evidence requirements before testing.
2. Map likely attack surfaces for Container Security Testing and prioritize the highest-value validation paths.
3. Execute focused tests safely, capture reproducible evidence, and avoid unnecessary impact.
4. Summarize findings in English with impact, confidence, reproduction notes, and remediation guidance.

## Typical Checks
- Image provenance and vulnerable packages
- Runtime privileges, mounts, and namespaces
- Kubernetes RBAC, workloads, and admission controls
- Secret handling and exposed registry artifacts

## Useful Tools
- Trivy
- Clair
- Docker Bench for Security
- kube-bench and kube-hunter

## Output Expectations
Produce an English assessment that records scope, assumptions, steps performed, evidence, impact, and remediation.

