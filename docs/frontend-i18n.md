# Frontend I18n Notes

The frontend keeps language strings under `web/static/i18n/`.

## Files
- `en-US.json`: primary English interface strings
- `zh-CN.json`: mirrored language pack, now aligned to English text for this repository-wide translation pass
- `web/static/js/i18n.js`: runtime loader and locale switching logic

## Recommended Workflow
1. Add new interface keys in `en-US.json`.
2. Keep keys synchronized across locale files.
3. Prefer semantic keys grouped by feature area instead of page-specific ad hoc strings.
4. Rebuild or refresh the frontend after changing locale files.
