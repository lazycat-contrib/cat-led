# RTC power schedules

The main toolbar provides a separate entry for LazyCat ROLE_ADMIN users. Every API
mutation and shutdown execution checks the current role through the LazyCat Users
API. Lookup failures deny access. OIDC sessions are signed; legacy plaintext
identity and role cookies are not accepted.

One device shares one power plan, either once or on selected weekdays. Shutdown
and RTC wake are independently selectable; disabling both cancels the plan. Nil
switches preserve paired behavior for older persisted plans. Power operations
appear in the main task list with distinct timer/alarm icons and labels.
Weekly schedules use an IANA timezone. A wake time no later than the shutdown time
means the following day only when both operations are enabled. Single-operation
weekly plans run on the selected days without this offset. Scheduling requires
two minutes of notice; paired operations need at least five minutes between them. The existing SQLite database gains one
rtc_power_plan singleton table for the specification, phase, actor and errors.

The preparing state is committed before writing the RTC. Successful readback
allows the active state. Before shutdown, the creator role and alarm are checked
again; waiting_wake is committed before the LazyCat poweroff RPC. Ambiguous RPC
failures are never retried automatically. Recovery skips missed shutdowns.
Wake-only schedules never invoke shutdown. Shutdown-only schedules do not access
RTC unless they must remove a previously owned alarm. A shutdown_sent state prevents
shutdown-only replay after restart. The current creator role is checked before
rearming and recurrence. Recurring schedules program only their next alarm. Cancellation durably disables
shutdown first, then clears the alarm only if it still matches this plan.

Tests use fake RTC and shutdown implementations. Coverage includes authorization,
forged cookies, cross-site mutations, overnight/week boundaries, timezone changes,
SQLite persistence, RTC failures, cancellation and restart recovery. Real S5 wake
support must be verified on the target hardware with firmware wake enabled.

The RTC dialog uses two native accessible switches and only reveals selected
settings. Its paper-like entry and reverse exit use interruptible clip-path,
transform and opacity transitions (280 ms entry, 220 ms exit), with keyboard and
reduced-motion alternatives. Other dialogs keep their existing behavior.
