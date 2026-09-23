---
name: file-upload-testing
description: Methodology and checklist for file upload testing.
version: 1.0.0
---

# File Upload Testing

## Overview
Use this skill to assess file validation, storage design, content processing, and retrieval controls for upload workflows.

## Recommended Workflow
1. Confirm scope, targets, constraints, and evidence requirements before testing.
2. Map likely attack surfaces for File Upload Testing and prioritize the highest-value validation paths.
3. Execute focused tests safely, capture reproducible evidence, and avoid unnecessary impact.
4. Summarize findings in English with impact, confidence, reproduction notes, and remediation guidance.

## Typical Checks
- Extension, MIME, and magic-byte validation
- Storage isolation and direct execution risk
- Image, archive, and document parser abuse
- Access control and public retrieval exposure

## Useful Tools
- Burp Suite
- File format mutation tools
- Server-side content inspection utilities
- Controlled sample files

## Output Expectations
Produce an English assessment that records scope, assumptions, steps performed, evidence, impact, and remediation.

