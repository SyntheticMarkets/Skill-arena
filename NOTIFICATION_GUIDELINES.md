# Notification Guidelines

Status: Draft for review

Sprint: 1 - Product Identity & Design Foundation

This document defines how Skill Arena communicates events across toasts, banners, inbox notifications, emails, push notifications, wallet timelines, match summaries, and admin-visible notices.

## Notification Principles

Notifications should:

- Be precise.
- Teach the player what happened.
- Explain the next step.
- Match the seriousness of the event.
- Avoid fake urgency.
- Avoid casino language.
- Never invent activity or statistics.

Every notification should answer:

- What happened?
- Why does it matter?
- What can the player do next?

## Notification Types

### Success

Use for completed actions.

Example:

- "Deposit settled. Your live balance is now available for competition."

### Pending

Use when the system has accepted an action but the lifecycle is not complete.

Example:

- "Withdrawal requested. Treasury review is now pending."

### Warning

Use when the player should pay attention before continuing.

Example:

- "Ranked entry locked. Complete practice calibration to unlock live competition."

### Error

Use when an action failed and the player needs a recovery path.

Example:

- "Replay verification failed. The match remains under review."

### Educational

Use when the platform teaches a rule, result, or process.

Example:

- "Move blocked. The upper route is still locked by an active dependency."

### Competitive

Use when competition state changes.

Example:

- "Match found. Enter Arena to begin countdown."

## Wallet Notification Examples

Preferred:

- "Deposit pending. Provider confirmation has not arrived yet."
- "Deposit settled. Your live balance is now available for competition."
- "Withdrawal under review. Treasury approval is required before settlement."
- "Withdrawal rejected. Complete verification before requesting this amount."
- "Ledger complete. Your transaction history has been updated."

Avoid:

- "Payment successful."
- "Payment failed."
- "Processing."
- "Cashout done."

## Replay Notification Examples

Preferred:

- "Replay verified. No integrity flags found."
- "Replay under review. Timing validation did not complete."
- "Replay invalid. The puzzle hash does not match the match record."
- "Insight ready. Review the dependency that slowed your route."

Avoid:

- "Replay failed."
- "Invalid game."
- "Something went wrong."

## Match Notification Examples

Preferred:

- "Match found. Countdown begins when both competitors are ready."
- "Victory confirmed. Replay verification complete."
- "Defeat confirmed. Review replay insights before entering the next match."
- "Connection interrupted. Reconnect before the grace period ends."

Avoid:

- "You lost."
- "Winner!"
- "Hurry now!"

## Notification Anatomy

Recommended structure:

```text
Status sentence.
Reason or context.
Next action when useful.
```

Example:

```text
Withdrawal under review.
Treasury approval is required for this amount.
You can track each stage in Wallet.
```

## Email And Push Guidance

Emails may carry more explanation than toasts.

Push notifications should be short and actionable.

Security, wallet, and withdrawal notifications should prioritize clarity over excitement.

Competition notifications may carry more energy, but should remain precise.

## Approval Questions

1. Does each notification explain what happened?
2. Does it teach or guide when needed?
3. Does it avoid vague system language?
4. Does wallet messaging create trust?
5. Does competitive messaging create energy without fake urgency?
