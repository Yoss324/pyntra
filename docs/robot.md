# Robot Integration Guide

Pyntra can expose notifications and chat workflows through external messaging platforms such as DingTalk and Lark.

## Setup Checklist
1. Configure the robot provider in `config.yaml`.
2. Supply webhook or app credentials according to the selected platform.
3. Restart the service after saving configuration changes.
4. Validate connectivity from the web UI or the corresponding test endpoint.

## Operational Notes
- Keep secrets outside version control.
- Restrict robot capabilities to approved channels and scopes.
- Review long-lived connection logs when troubleshooting message delivery.
