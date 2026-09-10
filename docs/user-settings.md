# Account settings and upcoming reminders

## Scope and acceptance criteria

- The main toolbar opens a settings dialog. Notification configuration remains reachable there.
- `show_schedules` defaults to true and controls the complete LED/power task list, without changing execution. The toolbar, user information and light remain visible.
- `reminders_enabled` defaults to true; `reminder_minutes` defaults to 5 and accepts integers from 1 to 1440.
- Preferences belong to the authenticated user in the existing SQLite user preference record. Partial updates preserve unspecified settings, including bulb style. Existing rows receive defaults during migration.
- Reminders appear inside the open webpage during the configured lead window, even when the list is hidden. They describe scheduled actions, not confirmed execution.
- LED and legacy power tasks use server-calculated next times and the same visibility rules as the task list. RTC plans retain the existing administrator access boundary and use persisted next execution times.
- Refresh while visible, on returning to the tab and after edits; suppress expired or stale data and disabled/cancelled actions. Render names as text. No browser notification permission or background delivery.
- Chinese/English, light/dark, desktop and 375px layouts remain usable. Keyboard users can open, save and close settings.

## Implementation plan

1. Extend the Ent schema and regenerate with the Ent v0.14.6 generator (use the project's `golang.org/x/tools` version with Go 1.27). Add validated partial preference updates and tests for migration, user isolation and preservation.
2. Add a read-only upcoming schedule endpoint with server time. Test weekday, midnight, disabled and visibility cases. Reuse RTC status events from the existing power UI.
3. Add the settings dialog and a compact reminder strip beside the light area. Reuse existing styles and i18n. Persist only after successful saves; show load/save failures.
4. Verify with `node --test tests/*.test.cjs`, `go test -race ./...`, `go vet ./...`, `./build.sh`, and browser interactions. Review and push v0.4.0.

## Boundaries and code conventions

Use existing vanilla JavaScript, Go 1.25, Ent and SQLite; no new runtime dependency. Keep generated files reproducible. Derive identity from authenticated context, validate request fields server-side, and never change device schedules as a side effect of display settings. Tests live in `internal/handlers/*_test.go` and `tests/*.test.cjs`.

## Interaction details

The toolbar provides 160–180ms hover/press feedback only for a fine pointer. The settings dialog uses a 200ms opacity/scale transition for pointer activation; keyboard activation and reduced-motion mode remain immediate. Existing theme colors and icon assets are reused.
