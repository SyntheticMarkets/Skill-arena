# Onboarding Experience

Status: Draft for product approval

Sprint: Product onboarding redesign

This document defines the complete path from first launch to live competition. It is a product and UX contract only. It does not authorize frontend or backend implementation.

## Why This Exists

Onboarding must prove that Skill Arena is a competitive ecosystem before it asks a player to complete forms or trust it with money.

The journey should make the player think:

- I understand what this platform is.
- I can explore before committing.
- I can improve before risking money.
- I always know what is unlocked, what is locked, and why.
- My identity, matches, and money are handled transparently.

## Experience Sequence

```text
Boot Experience
  -> Arena Landing
  -> Guest Arena Hub
  -> Locked Feature Intent
  -> Login or Registration
  -> Email Verification
  -> Player Profile
  -> Financial Assessment
  -> Identity Verification
  -> Verification Pending
  -> Practice Access
  -> Verification Decision
  -> Live Competition Unlock
```

Registration should be triggered by meaningful player intent. A visitor may explore the Arena Hub first, but must authenticate before progress can be saved or protected features can be used.

## Phase 1: Boot Experience

Purpose:

- Establish Skill Arena as a destination, not a conventional website.
- Communicate precision, competition, and readiness in two to three seconds.

Sequence:

```text
Dark field
  -> Skill Arena mark appears
  -> A small deterministic arrow sequence resolves
  -> "Every Move Matters"
  -> "Arena initializing"
  -> Requested destination loads
```

Rules:

- First visit may show the full sequence once.
- Returning visits use a shortened transition.
- Deep links must preserve their destination after boot.
- Reduced-motion mode replaces movement with short fades.
- Boot must never hide network failure or extend perceived load artificially.
- Maximum target duration is three seconds on a healthy load.

## Phase 2: Arena Landing

Purpose:

- Explain the promise.
- Establish financial and competitive trust.
- Create desire to enter the ecosystem.

Story order:

```text
Experience
  -> Competition
  -> Progress
  -> Fairness
  -> Wallet trust
  -> Community
  -> Enter the Arena
```

Primary message:

`WHERE SKILL BECOMES VALUE.`

Supporting proof should communicate:

- Compete against real players.
- Improve through practice and replay insight.
- Enter verified ranked and tournament competition.
- Track every wallet state clearly.
- Earn recognition and eligible rewards through skill.

Primary action:

- Enter the Arena.

Secondary action:

- Log in.

The landing page must not present fake live activity, player counts, prize totals, or match statistics.

## Phase 3: Guest Arena Hub

Purpose:

- Let the visitor understand the platform before registration.
- Show the relationship between games, competition, progression, wallet, and trust.

Visible areas:

- Available game modules.
- Practice.
- Ranked.
- Tournaments.
- Leaderboards.
- Challenges.
- Wallet.
- Replays.
- Community.
- Support.

Guest behavior:

- Public game information, rules, public leaderboards, and trust explanations may be explored.
- Practice may offer a short untracked sample only if the approved game protocol supports it.
- Progress, personalized practice, ranked, tournaments, wallet, deposits, withdrawals, notifications, and private replays require authentication.
- Locked surfaces remain readable enough to explain their value; they are not decorative blurred boxes with no context.

When a guest selects a protected action, show a focused authentication gate that preserves intent.

Example:

```text
Selected action: Enter Ranked
Requirement: Create or access your competitor account
After authentication: Return to Ranked eligibility
```

## Phase 4: Account Creation

Account creation is a staged journey, not one long form.

### Step 1: Secure Account

Collect:

- Email.
- Password.
- Country of residence.
- Age confirmation.
- Terms acceptance.
- Privacy acceptance.
- Fair-play acknowledgement.

Primary action:

- Continue to email verification.

### Step 2: Email Verification

Required states:

- Email sent.
- Resend available.
- Token expired.
- Token already used.
- Account already verified.
- Verification complete.

The pending destination must survive verification so the player continues where they intended to go.

### Step 3: Player Identity

Collect:

- Nickname.
- Curated avatar.
- Display country where permitted.
- Timezone.
- Language.

Product rules:

- Verified legal identity is never publicly editable.
- Player-facing identity is nickname plus avatar.
- Initial avatars are curated platform assets, not uploaded photographs.
- Nickname policy, moderation, and change limits must be approved before implementation.
- Proposed nickname rule: one free change every 90 days; additional changes require a defined coin or support policy.

### Step 4: Financial Assessment

Purpose:

- Collect the information required to determine live-wallet eligibility and compliance review.

Candidate fields, subject to legal and provider approval:

- Employment status.
- Income range.
- Expected deposit range.
- Source of funds.
- Tax residency.
- Politically exposed person declaration.

This step must explain why each category is requested. It must not collect fields until legal, privacy, retention, and jurisdiction requirements are approved.

### Step 5: Identity Verification

Candidate evidence, subject to provider and jurisdiction rules:

- Identity document, passport, or driver's licence.
- Proof of address.
- Additional evidence when review requires it.

Required UX states:

- Not started.
- In progress.
- Submitted.
- Pending review.
- More information required.
- Rejected with reason and recovery action.
- Approved.
- Expired and renewal required.

## Phase 5: Verification Pending

Pending verification must not become a dead end.

Available:

- Practice games.
- Training.
- Daily challenges.
- Public leaderboards.
- Replay learning.
- Profile and security setup.
- Trust-status tracking.

Locked until approved:

- Live wallet activation.
- Deposits where policy requires approval.
- Withdrawals.
- Ranked cash competition.
- Paid tournament entry.
- Cash rewards settlement.

Every lock must display:

- The capability that is locked.
- The exact requirement.
- The current verification state.
- The next useful action.
- Where the player will return after completion.

## Phase 6: Live Competition Unlock

Approval is a meaningful progression event.

Notification:

```text
Identity verified.
Your live wallet and eligible live competitions are now unlocked.
Review your limits before entering your first live match.
```

Primary action:

- Review live eligibility.

The player should see:

- Verification complete.
- Live wallet state.
- Deposit and withdrawal limits.
- Eligible competition types.
- Trust Score effect.
- Security recommendations such as MFA.

## Trust Status

Trust Status is a persistent, player-readable model rather than a single KYC label.

It should cover:

| Area | Example State | Player Explanation |
|---|---|---|
| Email | Verified | Account ownership confirmed. |
| Player profile | Complete | Competitor identity created. |
| Financial assessment | Complete | Eligibility information received. |
| Identity | Pending review | Submitted evidence is being reviewed. |
| Address | More information required | A newer proof of address is required. |
| Live wallet | Locked | Unlocks after required verification completes. |
| Live competition | Locked | Practice remains available while review continues. |

The Trust Status view must never expose internal risk rules, fraud signals, or sensitive reviewer notes.

## Notification Center

The notification bell is the central action inbox for account, competition, wallet, and verification events.

Each notification must include:

- Category.
- State.
- Timestamp.
- Plain-language message.
- One destination.
- Read/unread state.

Deep-link examples:

| Notification | Destination |
|---|---|
| Address document required | Verification evidence step |
| Identity approved | Live eligibility summary |
| Withdrawal approved | Withdrawal timeline |
| Tournament starts soon | Tournament lobby |
| Daily challenge complete | Challenge result |
| Replay insight ready | Replay insight |

Notification links must restore the exact object or task, not open a generic dashboard.

## Arena Navigation

Primary navigation:

- Home.
- Games.
- Challenges.
- Leaderboards.
- Wallet.
- Replays.
- Community.
- Support.

Utility area:

- Notifications.
- Balance summary for authenticated eligible players.
- Help.
- Player avatar.

Public footer or support navigation:

- Contact.
- Terms and Conditions.
- Privacy.
- Responsible play and eligibility information where required.

## Profile Contract

The profile represents competitive identity.

Player-facing identity may show:

- Curated avatar.
- Nickname.
- Level and XP.
- Country where permitted.
- Trust status summary.
- Current rank.
- House.
- Achievements.
- Match history.

Legal identity and compliance evidence must never be exposed in public profile surfaces.

## Current Architecture Gap

The current backend supports a basic KYC state transition: submit, status, and admin approval. It does not currently provide the full product contract described here for:

- Financial assessment fields.
- Evidence upload and storage.
- Document type and expiry tracking.
- More-information and rejection workflows.
- Jurisdiction-specific requirements.
- Granular eligibility locks.
- Notification deep links for verification tasks.

Because the backend is feature frozen, these items must remain documented product requirements until a separately approved compliance-support sprint is authorized. High-fidelity mockups must label them as required contracts, not production-ready behavior.

## Approval Gate

Before implementation, product, legal/compliance, security, backend, and frontend owners must approve:

1. Whether guest practice is available and whether it is tracked.
2. Which actions require email, identity, address, or financial verification.
3. Which financial-assessment fields are legally required by jurisdiction.
4. Which KYC provider and evidence-retention model will be used.
5. The nickname and avatar policy.
6. Notification categories and deep-link destinations.
7. The exact live-competition unlock rules.
8. High-fidelity mockups for every state in this document.

No implementation should begin from this document alone.
