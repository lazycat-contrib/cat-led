# RTC power schedules

The main page provides a separate entry for LazyCat ROLE_ADMIN users. Every API
mutation and shutdown execution checks the current role through the LazyCat Users
API. Lookup failures deny access. OIDC sessions are signed; legacy plaintext
identity and role cookies are not accepted.

One device shares one shutdown/wake plan, either once or on selected weekdays.
Weekly schedules use an IANA timezone. A wake time no later than the shutdown time
means the following day. Scheduling requires two minutes of notice and at least
five minutes between shutdown and wake. The existing SQLite database gains one
rtc_power_plan singleton table for the specification, phase, actor and errors.

The preparing state is committed before writing the RTC. Successful readback
allows the active state. Before shutdown, the creator role and alarm are checked
again; waiting_wake is committed before the LazyCat poweroff RPC. Ambiguous RPC
failures are never retried automatically. Recovery skips missed shutdowns.
Recurring schedules program only their next alarm. Cancellation durably disables
shutdown first, then clears the alarm only if it still matches this plan.

Tests use fake RTC and shutdown implementations. Coverage includes authorization,
forged cookies, cross-site mutations, overnight/week boundaries, timezone changes,
SQLite persistence, RTC failures, cancellation and restart recovery. Real S5 wake
support must be verified on the target hardware with firmware wake enabled.
