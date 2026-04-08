# Limit Refresh UI Design

## Goal

Extend the tray submenu so each saved account shows:

- remaining limit percentage for the 5-hour window
- remaining limit percentage for the weekly window
- 5-hour window reset as a countdown
- weekly window reset as an absolute time
- silent background refresh of usage every 10 minutes

The feature must preserve the existing tray-first interaction model and avoid adding status noise during background refresh.

## UI

Inside `切换账号`, each account block uses this structure:

1. account row
2. `５ｈ` progress row
3. `５ｈ` reset row
4. `周窗` progress row
5. `周窗` reset row

Example:

```text
[当前] alice@example.com
５ｈ  ███████░░░  74%
    2h18m 后刷新
周窗  ██████░░░░  63%
    周三 09:00 刷新
```

Rules:

- 5-hour reset row uses countdown wording such as `2h18m 后刷新`
- weekly reset row uses absolute time such as `周三 09:00 刷新`
- account separator remains a blank row between accounts
- background auto-refresh is silent: no `同步中`, `刚更新`, or similar menu text

## Data

Usage snapshots must carry both usage percentages and both reset timestamps:

- `primary.resetsAt` -> 5-hour reset time
- `secondary.resetsAt` -> weekly reset time

`resetsAt` comes from Codex app-server `account/rateLimits/read`.

If a reset time is unavailable:

- the progress row still renders percentage
- the reset row renders `刷新时间未知`

If usage is unavailable:

- progress rows keep existing `暂不可用` behavior
- reset rows also render `暂不可用`

## Refresh behavior

- Keep the current immediate overview load on startup.
- Keep the current refresh after save, switch, delete, and manual menu refresh.
- Add a 10-minute ticker that refreshes usage in the background.
- The ticker must update the submenu data without overwriting the status line with automatic-refresh text.

## Formatting rules

- Countdown aims for short tray-safe text: `43m 后刷新`, `2h18m 后刷新`, `1d3h 后刷新`
- Weekly absolute time aims for compact but clear text:
  - same day: `今天 21:30 刷新`
  - within current week: `周三 09:00 刷新`
  - otherwise: `4/12 09:00 刷新`

## Testing

Add coverage for:

- parsing `resetsAt` into snapshot fields
- overview propagation of reset times
- submenu rows for ready and unavailable usage states
- countdown and weekly absolute time formatting
