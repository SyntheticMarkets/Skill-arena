# Game Economy

Status: Draft for review

Sprint: 1 - Product Identity & Design Foundation

This document defines the product relationship between practice play, live play, rewards, trust, deposits, withdrawals, tournament entries, prize pools, house rewards, and season rewards.

It is not an implementation document.

## Economy Purpose

The Skill Arena economy exists to support fair competition and meaningful progression.

It should never feel like gambling, pressure, or arbitrary reward distribution.

Every economic system must answer:

- Why does this exist?
- What behavior does it reward?
- How does the player understand it?
- How is it kept fair?
- How does it support trust?

## Economy Layers

```text
Practice
  -> Learning
  -> Replay insight
  -> Tutorial tips
  -> Daily challenge
  -> Practice progress
  -> Trust movement
  -> First deposit
  -> Live play
  -> Ranked
  -> Tournament entry
  -> Prize pool
  -> Rewards
  -> Withdrawal
  -> Return loop
```

## Practice Economy

Why it exists:

- Let players experience Skill Arena without financial risk.
- Prove the game loop before asking for money.
- Teach rules, replay, and improvement.

What it should reward:

- Completion.
- Learning.
- Clean play.
- Practice consistency.

What it should not do:

- Pretend practice rewards are withdrawable live money.
- Confuse practice balance with live balance.
- Pressure immediate deposit.

## Live Economy

Why it exists:

- Allow skill-based competition with real stakes.

Requirements:

- Live balance must be clearly separate from practice balance.
- Available balance must be separate from locked and pending funds.
- Every live movement must be auditable.
- Every financial operation must have an idempotent lifecycle.
- Players must understand fees, limits, verification, and settlement states.

## Rewards

Rewards should reinforce skill, trust, and return behavior.

Reward types may include:

- Challenge rewards.
- Ranked rewards.
- Tournament winnings.
- House rewards.
- Season rewards.
- Trust-based access unlocks.

Reward rules:

- Rewards must have clear eligibility.
- Rewards must show whether they are practice, locked, pending, or available.
- Rewards must be connected to verified outcomes.
- Rewards must not imitate gambling mechanics.

## Trust Score Relationship

Trust Score should influence access, not feel like a mystery score.

Trust can affect:

- Ranked eligibility.
- Tournament eligibility.
- Withdrawal review intensity.
- Limits.
- Challenge access.
- House progression.

Trust should be increased by:

- Verified fair play.
- Account security completion.
- Clean match history.
- Completed verification.
- Consistent replay-valid outcomes.

Trust should be protected by:

- Suspicious activity review.
- Replay integrity checks.
- AML/risk review.
- Device/session security.

## Deposits

Deposits should happen only after Skill Arena has earned enough player confidence.

Preferred pre-deposit journey:

```text
Practice match
  -> Replay insight
  -> Tutorial tips
  -> Daily challenge
  -> Practice progress
  -> Trust increase
  -> First deposit
```

Deposit product requirements:

- Explain why deposit is needed.
- Show payment method.
- Show provider session.
- Show pending state.
- Show verification and settlement.
- Show when funds become available.

## Withdrawals

Withdrawals are the strongest trust moment in the product.

Withdrawal product requirements:

- Show available balance.
- Show limits and verification requirements.
- Show pending state.
- Show AML/risk review when applicable.
- Show treasury approval.
- Show provider settlement.
- Show ledger completion.
- Explain rejection clearly.

The player should feel:

- "The platform is careful with money."

Not:

- "The platform is hiding my money."

## Tournament Entry And Prize Pools

Tournament economics must be transparent before entry.

Required product clarity:

- Entry requirement.
- Entry fee when applicable.
- Prize pool.
- Reward distribution.
- Bracket rules.
- Replay dispute rules.
- Withdrawal implications.
- Cancel/refund conditions.

Prize pools should feel prestigious, structured, and auditable.

## House Rewards

House rewards should create belonging and long-term motivation.

They should reward:

- Participation.
- Clean competition.
- Improvement.
- Contribution to house objectives.

They should not overpower individual skill competition.

## Season Rewards

Season rewards should create long arcs of progress.

Season economy should explain:

- Season duration.
- Ranking impact.
- Reward eligibility.
- Trust requirements.
- Tie-breakers.
- Claim process.
- Expiry or rollover rules when applicable.

## Economy Approval Questions

1. Does the economy prove value before asking for deposit?
2. Are practice and live balances impossible to confuse?
3. Does every reward have a clear reason?
4. Does Trust Score influence access transparently?
5. Are withdrawals designed as trust moments?
6. Can future games participate without changing the economy model?
