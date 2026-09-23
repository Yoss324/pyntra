# Tool Definitions

Each YAML file in this directory describes a tool wrapper that Pyntra can expose to agents.

## Expected Fields
- `name`
- `command`
- `short_description`
- `description`
- `parameters`

## Guidance
1. Keep descriptions concise and operational.
2. Document parameters in English.
3. Avoid unsafe defaults and clarify when free-form arguments are appended to the command.
