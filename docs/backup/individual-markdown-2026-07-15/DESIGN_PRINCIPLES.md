# Design Principles

Status: Draft for review

Sprint: 1 - Product Identity & Design Foundation

## Primary Principle

Every screen must answer:

What is the next action that gets this player into a match, improves their skill, or advances their competitive progress?

If the answer is unclear, the screen is not ready.

## Core Principles

### 1. One Primary Action Per Screen

Every page should have one dominant next step.

Examples:

- Landing: enter the Arena.
- Guest Arena Hub: explore or select a protected action.
- Dashboard: continue playing.
- Wallet: choose deposit or withdraw based on context.
- Arena Hub: choose a game module.
- Match Summary: replay, rematch, or queue again.

Secondary actions may exist, but they must not compete visually with the primary action.

### 1A. Preserve Player Intent

Authentication, email verification, identity checks, and eligibility gates must return the player to the action that started the flow.

Examples:

- A guest who selects Ranked returns to Ranked eligibility after authentication.
- A player who opens a document-required notification returns to the exact verification task.
- A player who completes a wallet requirement returns to the pending deposit or withdrawal.

Do not send every completed flow to a generic dashboard.

### 2. Progress Must Be Visible

Players should always understand what improved, unlocked, changed, or moved.

Progress signals may include:

- Skill improvement.
- Trust Score movement.
- XP and level.
- League or MMR movement.
- Challenge completion.
- Replay verification.
- Wallet settlement.
- Tournament advancement.

### 3. Competition Should Feel Alive

The platform should communicate that other players are present and active.

Use:

- Live activity.
- Queue status.
- Recent match outcomes.
- Leaderboard movement.
- Tournament countdowns.
- Rival/opponent context.

Do not use fake statistics. If real data is unavailable, omit the module.

### 4. Trust Must Be Designed, Not Claimed

Trust is created through clarity and proof.

Trust-building UI includes:

- Clear balance states.
- Pending/settled labels.
- Replay integrity status.
- Audit-style timelines.
- MFA/security prompts.
- Transparent error messages.
- Explicit transaction lifecycle.

Avoid vague labels such as "processing" when a more precise status exists.

### 5. The Platform Is Game-Agnostic

Maze Arena is the first game, not the platform identity.

Shared UI must support:

- Maze Arena.
- Memory Arena.
- Logic Arena.
- Pattern Arena.
- Reaction Arena.
- Future games.

No global navigation, dashboard, wallet, leaderboard, or tournament UI should assume Maze-specific mechanics.

### 6. Make Risk Understandable

Because Skill Arena supports live balances, the interface must clearly separate:

- Practice balance.
- Live balance.
- Available balance.
- Locked balance.
- Pending withdrawals.
- Rewards.
- Fees.

Money movement should always show state, reason, and next step.

### 7. Reward Focus, Not Noise

The arena should feel energetic, but not chaotic.

Motion, effects, and live activity should support:

- Anticipation.
- Feedback.
- Victory.
- Progress.
- Status changes.

They should not distract from gameplay, wallet actions, or security decisions.

### 8. Fast Feedback Everywhere

Every interaction should visibly respond.

Required states:

- Loading.
- Disabled.
- Hover/focus.
- Success.
- Error.
- Pending.
- Empty.
- Locked.
- Verified.

### 9. Explain Failure Constructively

Defeat, blocked moves, failed payments, rejected withdrawals, and verification errors should tell the player what happened and what to do next.

Bad:

- "Error."
- "Failed."

Good:

- "Withdrawal rejected: KYC approval required for this amount."
- "Move blocked: dependency still active."
- "Replay flagged: route timing was too fast for verification."

### 10. Build For Repetition

Players will see core screens many times.

Design should support repeated use:

- Scannable dashboards.
- Compact but clear wallet data.
- Quick queue entry.
- Persistent progress context.
- Minimal friction after first use.

### 11. Every Screen Must Teach

Skill Arena should constantly help players understand the system.

If a player loses, they should understand why.

If a player wins, they should understand why.

If a player withdraws, they should understand every stage of the process.

Teaching creates trust because players can see cause and effect instead of guessing.

Examples:

- Match summary explains route efficiency, mistakes, timing, and dependency decisions.
- Replay shows what changed the result, not only what happened.
- Wallet timelines explain provider, pending, review, settlement, and ledger stages.
- Ranked screens explain MMR movement, placement, promotion, and demotion.
- Trust Score screens explain which actions increased, limited, or protected access.
- Error states explain the next useful action instead of ending the flow.

## Page-Level Design Tests

Before a page is approved, ask:

1. Why does this page exist?
2. What is the primary action?
3. What progress does this page show?
4. What competitive context does this page create?
5. What trust signal does this page provide?
6. Does this work for future games?
7. Does it avoid fake or placeholder data?
8. Is the page premium, or does it feel like CRUD?
9. What does this screen teach the player?

## Motion Principles

Motion should be:

- Fast.
- Purposeful.
- Directional.
- tied to state changes.

Use motion for:

- Queue transitions.
- Match start.
- Success/failure.
- Balance state changes.
- Rank movement.
- Replay playback.

Avoid motion for:

- Decorative noise.
- Constant looping distractions.
- UI that must be read carefully.

## Accessibility Principles

The platform must be usable under competitive pressure.

Requirements:

- Clear focus states.
- Keyboard navigation.
- Sufficient contrast.
- Motion reduction support.
- Text that does not overlap.
- Error messages that do not rely on color alone.
- Touch targets suitable for mobile.

## Approval Gate

No implementation should begin until these principles are approved.
