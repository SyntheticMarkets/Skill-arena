# Design System Plan

Status: Draft for review

Sprint: 1 - Product Identity & Design Foundation

This document is planning only. It does not define final implementation code.

Before any implementation sprint, the product path must be:

```text
Product Documents
  -> Wireframes
  -> High-Fidelity UX Mockups
  -> Approval
  -> Implementation
```

The design system should not be implemented from text descriptions alone. Approved high-fidelity UX mockups must guide the actual visual execution.

## Design System Purpose

The design system must make Skill Arena feel like a premium competitive gaming platform and keep every future page aligned with the North Star:

Every session should leave the player feeling they made meaningful progress.

The system must support:

- Platform pages.
- Wallet and financial flows.
- Arena Hub.
- Maze Arena.
- Future games.
- Admin and treasury tools.

## Visual Direction

The visual language should communicate:

- Arena energy.
- Precision.
- Trust.
- Competition.
- Progress.

It should avoid:

- Casino styling.
- Generic SaaS dashboards.
- Toy-like game UI.
- Overloaded neon effects.
- One-note color palettes.

## Visual Inspirations

These references are not to be copied. They define the feeling Skill Arena should study before choosing colors, typography, spacing, motion, or component style.

### Chess.com

What to learn:

- Daily return loops.
- Skill improvement as habit.
- Match history and analysis.
- Casual entry with deep competitive mastery.

What not to copy:

- Casual visual softness if it weakens premium arena energy.

### FACEIT

What to learn:

- Competitive seriousness.
- Queue and matchmaking tension.
- Player status, ranking, and tournament identity.
- Esports credibility.

What not to copy:

- Density that makes onboarding feel intimidating.

### Riot Client

What to learn:

- Game launcher as destination.
- Strong mode selection.
- Event energy.
- Player identity and account progression.

What not to copy:

- Heavy franchise-specific art direction.

### Steam

What to learn:

- Library and hub mental model.
- Activity surfaces.
- Community proof.
- Durable account identity.

What not to copy:

- Store-first browsing behavior.

### Apple Wallet

What to learn:

- Financial clarity.
- Confidence through restraint.
- Transaction readability.
- Strong state hierarchy.

What not to copy:

- Minimalism that strips away competitive emotion.

### Reference Synthesis

Skill Arena should feel like:

- The competitive seriousness of FACEIT.
- The improvement loop of Chess.com.
- The destination quality of Riot Client.
- The ecosystem depth of Steam.
- The financial trust clarity of Apple Wallet.

It should not become a clone of any one reference.

## Color Strategy

Color roles should be semantic before decorative.

Required roles:

- Background.
- Surface.
- Elevated surface.
- Primary action.
- Secondary action.
- Success.
- Warning.
- Danger.
- Pending.
- Verified.
- Locked.
- Live balance.
- Practice balance.
- Rank/progression.

The palette should support future games without making Maze Arena the visual identity.

## Typography Strategy

Typography must support:

- Fast scanning.
- Competitive emphasis.
- Financial clarity.
- Dense admin data.
- Mobile readability.

Required scales:

- Display.
- Page title.
- Section title.
- Card title.
- Body.
- Caption.
- Data label.
- Numeric emphasis.

## Spacing And Grid

The layout system should support:

- App shell.
- Dashboard grids.
- Wallet banking flows.
- Game cards.
- Leaderboards.
- Tables.
- Match surfaces.
- Admin panels.

Spacing should prioritize clarity and repeat use over decorative whitespace.

## Component Inventory

Planned reusable components:

- App shell.
- Top navigation.
- Side navigation.
- Page header.
- Section header.
- Card.
- Stat tile.
- Progress meter.
- Button.
- Icon button.
- Input.
- Select.
- Checkbox.
- Toggle.
- Tabs.
- Segmented control.
- Table.
- Badge.
- Status pill.
- Alert.
- Toast.
- Dialog.
- Drawer.
- Tooltip.
- Empty state.
- Loading skeleton.
- Error panel.
- Success panel.
- Timeline.
- Stepper.
- Balance display.
- Transaction row.
- Game card.
- Queue status.
- Match summary block.
- Replay status block.

## State Requirements

Every interactive component must define:

- Default.
- Hover.
- Focus.
- Active.
- Disabled.
- Loading.
- Error.
- Success.

Financial components must additionally define:

- Pending.
- Settled.
- Rejected.
- Locked.
- Available.

Game components must additionally define:

- Waiting.
- Queued.
- Active.
- Victory.
- Defeat.
- Verified replay.
- Flagged replay.

## Motion Plan

Motion should be tied to product meaning:

- Match found.
- Countdown.
- Queue state.
- Rank movement.
- Trust Score change.
- Balance settlement.
- Replay playback.
- Victory/defeat.

Motion must support reduced-motion preferences.

## Icon Strategy

Icons should clarify:

- Play.
- Queue.
- Wallet.
- Deposit.
- Withdraw.
- Lock.
- Unlock.
- Shield/security.
- Replay.
- Trophy.
- Rank.
- Settings.
- Alert.
- Verified.

Icons should not replace labels where financial or security clarity is required.

## UI Library Structure

Future implementation target:

```text
frontend/components/ui
  app-shell
  navigation
  buttons
  forms
  feedback
  data-display
  wallet
  game
  replay
  layout
```

No components should be created until this plan is approved.

## Acceptance Criteria For Future Sprint 1 Implementation

When implementation begins after approval:

- Components must be reusable.
- Components must not contain Maze-only assumptions.
- Components must include all required states.
- Components must be documented by usage.
- Components must pass build/tests.
- Design review must confirm platform identity alignment.

## Approval Questions

1. Does this system support the product identity?
2. Does it support wallet trust and game energy at the same time?
3. Does it avoid looking like a generic SaaS CRUD app?
4. Does it support future games?
5. Does it make the next action clear?
