# Player Journey

Status: Draft for review

Sprint: 1 - Product Identity & Design Foundation

## North Star Journey

Every session should leave the player feeling they made meaningful progress.

The player journey must show progress before, during, and after matches.

## Complete Journey

```text
New Visitor
  -> Boot Experience
  -> Arena Landing
  -> Guest Arena Hub
  -> Protected Action Selected
  -> Registration
  -> Email Verification
  -> Player Profile
  -> Financial Assessment
  -> Identity Verification
  -> Verification Pending
  -> First Practice Match
  -> Match Summary
  -> Replay Insight
  -> Tutorial Tips
  -> Daily Challenge
  -> Practice Progress
  -> Trust Increase
  -> Live Competition Unlock
  -> Dashboard Progress
  -> First Deposit
  -> Ranked Queue
  -> Ranked Match
  -> Leaderboard Movement
  -> Tournament Entry
  -> Tournament Result
  -> Withdrawal
  -> Return Daily
```

The complete onboarding state and lock contract is defined in `ONBOARDING_EXPERIENCE.md`.

## Step Detail

### 1. New Visitor

Goal:

- Understand what Skill Arena is.
- Believe it is competitive, fair, and worth joining.

Emotion:

- Curious.

Primary action:

- Enter the Arena.

Trust signals:

- Skill-based positioning.
- Replay integrity.
- Secure wallet messaging.
- Real progression promise.

### 2. Boot, Landing, And Guest Arena Hub

Goal:

- Establish Skill Arena as a destination.
- Let the visitor understand games, progression, competition, wallet trust, and locked live features before registration.

Emotion:

- Curious, then interested.

Primary action:

- Explore the Arena Hub, then select a meaningful protected action.

Trust signals:

- Clear practice and live distinction.
- Public rules and leaderboards.
- Locked capabilities explain their requirements.
- No fake activity or statistics.

### 3. Registration

Goal:

- Create competitor identity.

Emotion:

- Excited.

Primary action:

- Submit account details.

Trust signals:

- Clear terms.
- Age verification.
- Privacy clarity.

### 4. Email Verification

Goal:

- Confirm account ownership.

Emotion:

- Reassured.

Primary action:

- Verify email.

Trust signals:

- Clear resend flow.
- Expired token handling.
- Already verified handling.

### 5. Player Profile, Financial Assessment, And Identity Verification

Goal:

- Create a player-facing competitor identity.
- Collect only approved eligibility and verification information.
- Keep Practice available while live eligibility is reviewed.

Emotion:

- Invested and informed.

Primary action:

- Complete the next required verification step or continue to Practice.

Trust signals:

- Legal identity is separated from public profile identity.
- Every requested field explains why it is needed.
- Verification status, locks, and next actions are explicit.

### 6. First Practice Match

Goal:

- Experience the core game without financial risk.

Emotion:

- Focused.

Primary action:

- Complete practice match.

Progress signal:

- First completion.
- Route efficiency.
- Replay available.

### 7. Match Summary

Goal:

- Understand outcome and improvement.

Emotion:

- Informed.

Primary action:

- Queue again or view replay.

Progress signal:

- Skill metric change.
- XP.
- Trust effect.
- Best route comparison.

### 8. Replay Insight

Goal:

- Learn from the match.

Emotion:

- Insightful.

Primary action:

- Try again.

Trust signals:

- Replay verified.
- Integrity flags visible when relevant.

### 9. Tutorial Tips

Goal:

- Convert the replay into useful learning.
- Show the player that improvement is visible and achievable.

Emotion:

- Encouraged.

Primary action:

- Apply one tip in another practice or training run.

Progress signal:

- Suggested skill focus.
- Route decision explanation.
- Timing or accuracy improvement target.
- "Try this next" guidance.

Trust signals:

- The platform explains outcomes without blaming the player.
- The player can improve before spending money.

### 10. Daily Challenge

Goal:

- Give the player a structured reason to play again before live competition.

Emotion:

- Motivated.

Primary action:

- Start daily challenge.

Progress signal:

- Challenge completion.
- Personal best.
- Skill streak.
- Practice reward or trust-safe reward when allowed.

Trust signals:

- Clear rules.
- No financial pressure.
- Replay-backed result when relevant.

### 11. Practice Progress

Goal:

- Prove the player enjoys the core loop before asking for a first deposit.

Emotion:

- Confident.

Primary action:

- Continue practice or unlock ranked eligibility.

Progress signal:

- Practice level.
- Accuracy trend.
- Completion trend.
- Replay insight history.

Trust signals:

- Practice progress and practice balance are separate from live balance.
- The player understands the game enough to make an informed deposit decision.

### 12. Trust Increase

Goal:

- Show that fair play unlocks access.

Emotion:

- Rewarded.

Primary action:

- Continue playing or complete verification.

Progress signal:

- Trust Score movement.
- Trust tier.
- New eligibility.

### 13. Live Competition Unlock

Goal:

- Turn verification approval into a clear progression event.

Emotion:

- Trusted and ready.

Primary action:

- Review live eligibility and limits.

Progress signal:

- Live wallet status.
- Eligible competition types.
- Trust status movement.

### 14. Dashboard Progress

Goal:

- Give the player a clear next step.

Emotion:

- Motivated.

Primary action:

- Continue playing.

Progress signal:

- Daily progress.
- Season progress.
- Wallet snapshot.
- Leaderboard preview.

### 15. First Deposit

Goal:

- Fund live play safely.

Emotion:

- Confident.

Primary action:

- Choose payment method.

Trust signals:

- Provider session.
- Pending/settled state.
- Available vs locked balance.
- Fees and limits.

Important product rule:

- The first deposit should come after practice play, replay insight, tutorial guidance, daily challenge exposure, practice progress, and visible trust movement.
- Skill Arena should earn the deposit by proving the product is fair, enjoyable, and understandable.

### 16. Ranked Queue

Goal:

- Enter competitive play.

Emotion:

- Anticipation.

Primary action:

- Join queue.

Progress signal:

- League/MMR context.
- Estimated match quality.

### 17. Ranked Match

Goal:

- Compete under pressure.

Emotion:

- Focus.

Primary action:

- Play.

Trust signals:

- Backend-owned opponent progress.
- Replay-ready match.
- Clear rules.

### 18. Leaderboard Movement

Goal:

- See competitive status change.

Emotion:

- Ambition.

Primary action:

- Queue again.

Progress signal:

- Rank movement.
- Country/global/season position.

### 19. Tournament Entry

Goal:

- Join a higher-prestige competition.

Emotion:

- Prestige.

Primary action:

- Register.

Trust signals:

- Prize pool.
- Bracket rules.
- Entry requirements.
- Replay dispute support.

### 20. Withdrawal

Goal:

- Convert earned balance into payout.

Emotion:

- Trust.

Primary action:

- Request withdrawal.

Trust signals:

- Pending state.
- AML/review state.
- Treasury approval.
- Settlement timeline.

### 21. Return Daily

Goal:

- Continue the progress loop.

Emotion:

- Momentum.

Primary action:

- Start daily challenge or ranked queue.

Progress signal:

- Daily streak.
- New challenges.
- Season movement.

## Journey Risks

Potential failure points:

- Visitor does not understand skill-based competition.
- Registration asks too much too early.
- Guest exploration becomes a dead-end marketing preview instead of teaching the platform.
- Protected actions forget the player's intended destination after authentication.
- Verification pending blocks Practice and causes abandonment.
- KYC locks do not explain the exact requirement or recovery action.
- First deposit is requested before the player has experienced enough value.
- Wallet feels like a generic balance table.
- Dashboard becomes a data dump.
- Arena Hub hides the best next action.
- Losing feels empty.
- Replay is hard to understand.
- Withdrawal status feels vague.

## Journey Approval Questions

1. Does every step make the player feel progress?
2. Does the first practice match happen early enough?
3. Does practice, replay, tutorial guidance, daily challenge, and practice progress earn the first deposit?
4. Does the journey work for non-Maze future games?
5. Does the return loop feel strong enough for daily play?
