# Skill Arena: Phase 1 & 3 Requirements Extraction

## Executive Summary

This document extracts key requirements from Phase 1 and Phase 3 PDFs, organized by component with core features/APIs, database models, user workflows, and business logic for each system.

---

# PHASE 1: FOUNDATION & GAME DESIGN (Launch: Maze Arena)

## Core Constitutional Principles (Non-Negotiable)
1. **Skill Determines Outcomes** - Pure skill-based competition
2. **Every Live Match Is Replayable** - Full replay verification capability
3. **Every Token Is Auditable** - Complete financial traceability
4. **Every Challenge Is Verifiable** - All results can be validated
5. **Server Authority** - Client trust level = ZERO for business logic
6. **No Reward May Exceed Treasury Reserves** - Financial sustainability
7. **Infinite Progression** - No caps on advancement
8. **Fair Seasonal Competition** - Resets and seasonal mechanics
9. **Sustainability First** - Long-term viability over short-term profit

---

# PHASE 1: COMPONENT SPECIFICATIONS

## 1. USER SYSTEM & AUTHENTICATION

### Core APIs Needed:
```
POST /auth/register         → Register new user
POST /auth/verify-email     → Verify email
POST /auth/login            → Authenticate user
POST /auth/mfa-setup        → Enable MFA
POST /auth/mfa-verify       → Verify MFA token
POST /auth/refresh-token    → Get new access token
POST /identity/kyc-submit   → Submit KYC verification
GET  /identity/kyc-status   → Check KYC status
POST /devices/fingerprint   → Register device
```

### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `users` | User identity | user_id, email, password_hash, kyc_status, verified_date, created_at |
| `user_profiles` | User info | user_id, avatar, country, username, bio |
| `user_sessions` | Active sessions | user_id, session_token, refresh_token, expires_at |
| `user_devices` | Device tracking | user_id, device_fingerprint, device_name, os, browser, last_seen |
| `kyc_records` | Identity verification | user_id, verification_provider, status, document_type, verified_date |

### User Workflows:
```
1. Registration Flow:
   - Email registration → Email verification → Profile setup → 
   - KYC submission → Admin verification → Account activated

2. Login Flow:
   - Email + password → JWT issued → MFA challenge (if enabled) → 
   - Session created → Ready for API calls

3. Account Escalation:
   - Basic account → Verified account (high-value withdrawals) → 
   - KYC approved (withdraw limits lifted)

4. Security Workflow:
   - Enable MFA → Download recovery codes → Confirm MFA works
```

### Business Logic:
- Password: Min 12 chars, uppercase, numbers, symbols required
- Email verification required before wallet activation
- KYC mandatory for withdrawals > USD $500
- MFA enforced for withdrawals
- Device fingerprinting prevents account sharing
- Session timeout: 30 days (refresh tokens)

---

## 2. WALLET & TOKEN SYSTEM

### Core APIs Needed:
```
GET  /wallet/balance          → Get current balance
POST /wallet/deposit          → Initiate deposit
POST /wallet/withdrawal       → Initiate withdrawal
POST /wallet/lock-tokens      → Reserve tokens for match
POST /wallet/unlock-tokens    → Release reserved tokens
GET  /wallet/transactions     → Transaction history
GET  /wallet/available        → Available balance (not locked)
```

### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `wallets` | User wallets | wallet_id, user_id, wallet_type, balance, status, created_at |
| `wallet_types` | Wallet categories | LIVE_WALLET, DEMO_WALLET, LOCKED_WALLET, BONUS_WALLET, SYSTEM_WALLET |
| `transactions` | All movements | transaction_id, wallet_id, amount, type, reference_id, status, timestamp |
| `ledger_entries` | Double-entry ledger | entry_id, transaction_id, debit_account, credit_account, amount, balance_after |
| `wallet_audit` | Immutable audit log | audit_id, wallet_id, previous_balance, new_balance, reason, timestamp |

### User Workflows:
```
Deposit Workflow:
1. Player clicks "Deposit"
2. Select payment method and amount (min USD $10)
3. Redirected to payment provider
4. Payment provider confirms transaction
5. System receives webhook confirmation
6. Ledger entry created (debit: Bank, credit: Live Wallet)
7. Player receives tokens in Live Wallet
8. Available immediately for play

Withdrawal Workflow:
1. Player initiates withdrawal
2. System verifies: account verified, wallet balance, limits
3. AML screening performed
4. Admin manual review (if high value)
5. Withdrawal approved
6. Ledger entry created (debit: Live Wallet, credit: Bank Account)
7. Bank transfer initiated
8. Player receives funds in 1-3 business days

Match Entry Workflow:
1. Player enters match (10 tokens)
2. System locks tokens (debit: Live Wallet, credit: Locked Wallet)
3. Match occurs
4. Match completes - tokens unlocked
5. Ledger entries: Prize credited or fee taken
```

### Business Logic:
- **Wallet Types:**
  - `LIVE_WALLET`: Real money, can withdraw
  - `DEMO_WALLET`: Practice tokens, no value, can't transfer
  - `LOCKED_WALLET`: Reserved for active matches
  - `BONUS_WALLET`: Promotional tokens, withdrawal restrictions
  - `SYSTEM_WALLET`: Treasury system account

- **Balance Calculation:**
  ```
  Available Balance = Live Wallet - Locked Wallet - Pending Withdrawals
  ```

- **Minimum Deposit:** USD $10 (100 tokens at 1:10 rate)
- **Withdrawal Limits:**
  - Unverified: None (blocked unless KYC passes)
  - Verified: USD $50,000/day, USD $500,000/month
  - Enhanced: No limit

- **Fees:**
  - Deposits: 2.5% (absorbed by platform)
  - Withdrawals: 1% (charged to player)
  - Match entry: Included in PvP fee

---

## 3. PROGRESSION SYSTEMS (5 Independent)

### Core APIs Needed:
```
GET  /progression/xp         → Get XP level and prestige
GET  /progression/elo        → Get skill rating
GET  /progression/league     → Get league rank
GET  /progression/house      → Get house reputation
GET  /progression/legacy     → Get legacy points
POST /progression/award-xp   → Award XP (internal only)
```

### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `progression_xp` | Level system | user_id, current_level, total_xp_earned, prestige_level, never_reset |
| `progression_elo` | Skill rating | user_id, current_rating, matches_played, wins, losses, k_factor |
| `progression_league` | League rank | user_id, league_tier, rank_in_league, promotion_points, season_id |
| `progression_house_reputation` | House challenges | user_id, reputation_score, challenges_completed, win_rate |
| `progression_legacy` | Lifetime tracker | user_id, total_legacy_points, contribution_tier, never_reset |
| `prestige_milestones` | Prestige unlocks | prestige_level, xp_required, rewards_given |

### Progression Types:

#### 1. XP LEVEL (Infinite, Never Resets)
- **Unlock house tiers:** XP determines available house challenge tiers
- **XP Sources:**
  - Complete PvP match: +10 XP
  - PvP victory: +50 XP
  - House challenge success: +25 XP per tier
  - Tournament participation: +100 XP
  - Seasonal achievements: +500 XP
- **No cap** - progression continues indefinitely
- **Prestige System:**
  - Unlock after XP milestones (e.g., 100k XP = Prestige I)
  - No upper limit on Prestige
  - Permanent badge on profile
  - Cosmetic rewards at each level

#### 2. SKILL RATING (ELO-Based)
- **Match-based rating** reflecting competitive skill
- **Initial rating:** 1200 (for all new players)
- **Formula:** `New_Rating = Old_Rating + K_Factor × (Actual_Result - Expected_Result)`
  - K_Factor = 32 (standard competitive)
  - Actual_Result = 1 (win) or 0 (loss)
  - Expected_Result = calculated from opponent rating
- **Rating floors:** Minimum 1000 (no negative ratings)
- **Used for:** Matchmaking, league placement, tournament eligibility

#### 3. LEAGUE RANK (Seasonal, Resets)
- **League Tiers:** Bronze, Silver, Gold, Platinum, Diamond, Elite, Legend
- **Rank within tier:** 1-100
- **Promotion/Demotion:** Based on season points
- **Resets:** January 1st each season (mid-season optional)
- **Permanent progression:** Achievement for reaching each tier recorded in stats

#### 4. HOUSE REPUTATION (Separate System)
- **Earned from:** House challenge completions only
- **Score increases:** When winning house challenges
- **Unlocks:** Higher house tiers (Bronze → Silver → Gold, etc.)
- **Never impacts:** PvP rankings or seasonal standings
- **Win rate tracking:** Monitor individual player success vs house

#### 5. LEGACY POINTS (Lifetime, Never Resets)
- **Represents:** Lifetime contribution to platform
- **Sources:**
  - Seasonal participation: +10 points
  - Tournament success: +50 per placement
  - House challenge completion: +5 per tier
  - PvP activity: +1 per match
  - Seasonal achievements: +100 per achievement
- **Purpose:** Long-term prestige marker
- **Hall of Fame:** Top 1000 legacy point holders get recognition

### Business Logic:
```
XP_Level = floor(Total_XP_Earned / 1000)
Prestige_Level = floor(XP_Level / 100)
Available_House_Tiers = tiers where min_xp <= player_xp_level
Matchmaking_Rating = current_elo_rating
```

---

## 4. MATCH & GAMEPLAY SYSTEM

### Core APIs Needed:
```
POST /matches/create          → Create new match
GET  /matches/{match_id}      → Get match state
POST /matches/{match_id}/move → Submit player move
POST /matches/{match_id}/complete → Mark challenge complete
GET  /matches/{match_id}/replay  → Get replay data
GET  /matches/history         → Player match history
```

### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `matches` | Match metadata | match_id, match_type, player_1_id, player_2_id, maze_id, status, created_at, ended_at |
| `match_participants` | Player match data | match_id, user_id, entry_fee, final_result, completion_time, verified |
| `match_pots` | Financial tracking | match_id, total_pot, platform_fee, house_edge, winner_prize, loser_prize |
| `challenge_state` | Live game state | match_id, current_state_json, moves_validated, lives_remaining, completion_percent, last_update |
| `match_movements` | Move audit trail | movement_id, match_id, user_id, move_type, move_data, server_validated, timestamp |
| `replay_data` | Full replay recording | replay_id, match_id, compressed_movements, verification_hash, created_at |

### Match Types:

#### 1. RANKED QUEUE
- **Impact:** Affects ELO, league rank, seasonal standing
- **Matchmaking:** ELO ± 200 rating points
- **Entry fee:** 10 tokens (configurable)
- **Rewards:** Based on win/loss and ELO difference

#### 2. CASUAL QUEUE
- **Impact:** None (no ranking changes)
- **Entry fee:** 5 tokens (configurable)
- **Purpose:** Practice without penalty
- **Rewards:** Reduced, flat-rate

#### 3. FRIEND CHALLENGE
- **Opponent:** Specific player invite
- **Entry fee:** Configurable (both must agree)
- **Impact:** Custom (can be ranked or casual)
- **Privacy:** Match not shown on leaderboards

#### 4. CROSS-LEAGUE CHALLENGE
- **Opponent:** Different league tier
- **Adjustments:** ELO adjustment factors applied
- **Entry fee:** Variable based on league difference
- **Purpose:** Competitive fun without matchmaking constraints

#### 5. TOURNAMENT MATCH
- **Controlled by:** Tournament system
- **Entry fee:** Paid at tournament entry
- **Impact:** Tournament standing only
- **Rewards:** Tournament prize pool

### Match Lifecycle:

```
1. CREATION PHASE
   - Player selects queue type and entry amount
   - Server validates:
     * Account status (not banned)
     * Wallet balance ≥ entry fee
     * League eligibility (not above max or below min)
     * KYC status (verified if high value)
     * Device verified (not flagged)
   - If valid: Match created, entry fee LOCKED
   - Matchmaking begins

2. MATCHMAKING PHASE
   - Server finds opponent with similar rating
   - Timeout: 30 seconds (if no match, auto-refund)
   - Once paired: Both players notified
   - Both have 10 seconds to accept/decline

3. CHALLENGE GENERATION PHASE
   - Server generates shared maze seed
   - Both players receive SAME maze
   - Seed stored in match record (immutable)
   - Verification hash created

4. MATCH START PHASE
   - Both players see identical starting maze
   - Timer synchronized across both clients
   - Lives counter initialized
   - Match status: IN_PROGRESS

5. GAMEPLAY PHASE
   - Every move sent to server
   - Server validates move:
     * Is move in valid path? 
     * Is player alive?
     * Did movement comply with maze rules?
     * If invalid: Move rejected, client notified
   - Client renders authorized moves only
   - Progress tracked server-side

6. COMPLETION PHASE
   - First player to reach exit wins
   - OR timer expires (current leader wins)
   - OR both players fail (draw or lowest-loss)
   - Server marks match: COMPLETED

7. RESULTS PHASE
   - Server calculates results:
     * Winner determined
     * Match duration calculated
     * ELO changes calculated
     * Prize pool distributed
     * Ledger entries created
   - Players notified immediately
   - Replay data saved and verified
```

### Match Entry Validation (Server-Side Only):
```python
def validate_entry(player_id, match_type, entry_fee):
    # All checks MUST happen server-side
    account = get_account(player_id)
    
    # 1. Account Status
    if account.banned or account.suspended:
        return False, "Account restricted"
    
    # 2. Wallet Balance
    available = get_available_balance(player_id)
    if available < entry_fee:
        return False, "Insufficient balance"
    
    # 3. League Eligibility
    player_league = get_league_tier(player_id)
    if match_type == "ranked" and not is_eligible_for_ranked(player_league):
        return False, "League restrictions"
    
    # 4. KYC Status
    if entry_fee > 100 and not account.kyc_verified:
        return False, "KYC required"
    
    # 5. Tournament Qualification
    if match_type == "tournament":
        if not is_tournament_qualified(player_id):
            return False, "Tournament eligibility not met"
    
    return True, "Validated"
```

### Prize Pool Calculation:

```
Example: 
  Player A entry = 10 tokens
  Player B entry = 10 tokens
  
  Total Pot = 20 tokens
  Platform Fee = 10% × 20 = 2 tokens (goes to Platform Revenue Reserve)
  Actual Prize Pool = 20 - 2 = 18 tokens
  
  If Player A wins:
    Player A receives: 18 tokens
    Player B receives: 0 tokens
    
  If draw/mutual loss (both fail):
    Split 18 tokens: 9 each
```

### Business Logic:
- **Platform Fee:** 10% of entry pot
- **ELO Adjustment:** K=32, consider rating difference
- **Replay Storage:** All replays stored for 90 days minimum
- **Dispute Period:** 24 hours for players to dispute

---

## 5. HOUSE CHALLENGE ENGINE

### Core APIs Needed:
```
GET  /house/tiers            → Get available house tiers
POST /house/challenge        → Generate new house challenge
POST /house/submit           → Submit challenge completion
GET  /house/history          → Get challenge history
GET  /house/analytics        → Get personal vs house stats
```

### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `house_tiers` | Tier definitions | tier_id, tier_name, min_xp_level, min_elo, cost_per_attempt, reward_multiplier, win_rate_target |
| `house_challenges` | Challenge instances | challenge_id, player_id, tier_id, maze_id, difficulty_score, seed, created_at, expires_at |
| `house_results` | Outcome tracking | result_id, challenge_id, player_id, status (win/loss), completion_time, payout, treasury_impact |
| `house_analytics` | Player statistics | user_id, challenges_completed, win_rate, total_payout, avg_completion_time |

### House Tiers (Progressive):

| Tier | Min XP | Min ELO | Entry Fee | Reward | House WR Target |
|------|--------|---------|-----------|--------|-----------------|
| Bronze | 0 | 1200 | 5 | 7.5 | 65% |
| Silver | 10k | 1400 | 15 | 22.5 | 65% |
| Gold | 30k | 1600 | 50 | 75 | 65% |
| Platinum | 75k | 1800 | 150 | 225 | 65% |
| Elite | 150k | 2000 | 500 | 750 | 65% |
| Legend | 300k | 2200 | 1500 | 2250 | 65% |

### House Challenge Generation:

```
Challenge_Seed = Hash(
    player_id,
    timestamp,
    difficulty_tier,
    server_secret_salt,
    randomization_nonce
)

Challenge_ID = SHA256(Challenge_Seed)

Uniqueness_Check:
    IF Challenge_ID exists in database:
        Regenerate with new nonce
    ELSE:
        Store and use
```

### House Challenge Unlock Requirements:

```
def can_access_tier(player_id, tier):
    player_xp = get_xp_level(player_id)
    player_elo = get_elo_rating(player_id)
    player_reputation = get_house_reputation(player_id)
    
    # All requirements must be met
    return (
        player_xp >= tier.min_xp_level AND
        player_elo >= tier.min_elo AND
        player_reputation >= tier.min_reputation
    )
```

### House Challenge Features:

- **Unique for Each Player:** No two players get identical challenges
- **Deterministic:** Same seed always generates same maze (for verification)
- **Difficulty Calibrated:** Based on player skill model
- **Adaptive:** Win rate monitored, difficulty adjusted dynamically
- **Verifiable:** All results auditable via replay
- **Profitable for Platform:** House edge targets ~65% win rate

### House Win Rate Model (Adaptive):

```
Adaptive House Probability:
  - Target range: 60-70% house win rate
  - Operational target: 65%
  - Monitor daily completion rates

Dynamic Difficulty Adjustment:
  IF player_success_rate > 70%:
      Increase difficulty (more traps, tighter timer)
  ELSE IF player_success_rate < 60%:
      Decrease difficulty (more time, simpler layouts)
  ELSE:
      Maintain current difficulty

House Fairness Principle:
  - Advantage comes from difficulty design, not impossible conditions
  - Highly skilled players CAN win even highest tiers
  - No "rigged" mechanics, all mathematically verifiable
```

### Business Logic:
- **Treasury Protection:** Only payout if reserves sufficient
- **Daily Limits:** Player can do unlimited house challenges
- **Reputation Impact:** Winning increases house reputation score
- **XP Rewards:** +25 XP per tier for completion
- **Season Tracking:** House performance affects seasonal achievements

---

## 6. SEASONAL SYSTEM

### Core APIs Needed:
```
GET  /seasons/current        → Get current season info
GET  /seasons/{season_id}    → Get specific season
GET  /seasons/points         → Get player season points
POST /seasons/claim-reward   → Claim season rewards
GET  /seasons/pass-status    → Get pass info
POST /seasons/pass-purchase  → Buy premium pass
```

### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `seasons` | Season metadata | season_id, season_number, start_date, end_date, theme, status, total_fund_allocated |
| `season_points` | Player standings | user_id, season_id, points_earned, rank_in_season, tier_earned |
| `season_achievements` | Seasonal badges | achievement_id, season_id, requirement_type, point_value |
| `season_passes` | Pass purchases | pass_id, user_id, season_id, pass_type (free/premium), purchased_date |
| `season_rewards` | Reward definitions | reward_id, season_id, point_threshold, reward_amount, reward_type |
| `season_rewards_earned` | Claimed rewards | earning_id, user_id, season_id, reward_id, claimed_date |

### Season Structure:

```
Season Duration: 90 days (configurable)
Example:
  Season 1: Jan 1 - Mar 31
  Season 2: Apr 1 - Jun 30
  Season 3: Jul 1 - Sep 30
  Season 4: Oct 1 - Dec 31

At Season End:
  - Season Points RESET to 0
  - League Rank RESET (if soft reset enabled)
  - XP, Prestige, Legacy Points REMAIN (permanent)
  - Season achievements locked (can't earn more)
  - Rewards distributed
  - New season begins immediately
```

### Season Points System:

**Sources of Season Points:**
- PvP Victory: +10 points (base, scales with opponent rating)
- House Challenge Success: +5 points (per tier)
- Tournament Participation: +50 points
- Seasonal Objective: +100 points (e.g., "win 10 matches")
- Seasonal Achievement: +500 points (major milestones)

**Season Points Reset:**
```
def end_season(season_id):
    # Reset season-specific data
    UPDATE season_points SET points_earned = 0
    UPDATE season_pass SET status = 'expired'
    UPDATE season_achievements SET earned = False
    
    # DO NOT RESET
    UPDATE progression_xp  -- Keep XP
    UPDATE progression_legacy  -- Keep legacy
    UPDATE progression_prestige  -- Keep prestige
    
    # Archive current season
    ARCHIVE season_id
```

### Season Pass System:

**Free Pass:**
- Available to all players
- Rewards: +50% XP on all activities
- No cost
- All progress visible

**Premium Pass:**
- Cost: TBD (e.g., 50 tokens or USD $5)
- Rewards: +100% XP, +50% Season Points, exclusive cosmetics
- Purchasable anytime
- Refund if not used (first 7 days)

### Season Rewards:

```
Rewards Funded By:
  1. Season Fund Reserve (primary)
  2. Tournament Revenue Allocation (10% of tournament fees)
  3. Platform Revenue Allocation (5% of platform fees)

Distribution Examples:
  Top 10 players: 1000 tokens each
  Top 50 players: 500 tokens each
  Top 100 players: 250 tokens each
  Top 1000 players: 50 tokens each

Sustainability Rule:
  Total_Rewards_Available = Season_Fund_Budget
  IF rewards_needed > budget:
      Scale rewards down proportionally
      Increase season fund next quarter
```

### Business Logic:
- **Reward Funding:** Must never exceed Season Fund balance
- **Tier Achievements:** Reaching league tiers = seasonal achievement
- **Pass Progression:** Daily login streaks earn bonus points
- **Leaderboard Visibility:** Top 100 always displayed publicly

---

## 7. TOURNAMENT SYSTEM

### Core APIs Needed:
```
GET  /tournaments            → List active tournaments
GET  /tournaments/{id}       → Tournament details
POST /tournaments/{id}/enter → Register for tournament
GET  /tournaments/{id}/bracket  → View brackets
GET  /tournaments/{id}/results  → Get final results
POST /tournaments/{id}/claim  → Claim rewards
```

### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `tournaments` | Tournament metadata | tournament_id, tournament_type, entry_fee, max_participants, prize_pool, duration, start_date, status |
| `tournament_participants` | Registrations | participant_id, tournament_id, user_id, entry_fee_locked, qualification_met, final_placement |
| `tournament_brackets` | Match structure | bracket_id, tournament_id, round_number, seed_1, seed_2, winner, status |
| `tournament_matches` | Match results | match_id, tournament_id, bracket_id, player_1, player_2, result, match_data |
| `tournament_rewards` | Prize pool | reward_id, tournament_id, placement, reward_amount, reward_status |

### Tournament Hierarchy:

| Type | Frequency | Entry Fee | Duration | Max Players | Prize Pool | Prestige |
|------|-----------|-----------|----------|-------------|------------|----------|
| Daily | Every 24h | 5 tokens | 24h | Unlimited | Dynamic | Low |
| Weekly | Every 7d | 25 tokens | 7d | 256 | Dynamic | Medium |
| Monthly | Every 30d | 100 tokens | 30d | 512 | Fixed | High |
| Seasonal | Every 90d | 500 tokens | 90d | 1000 | Large | Very High |
| World | Annual | 1000 tokens | 30d | 2000 | Massive | Max |

### Tournament Qualification:

```
def check_tournament_eligibility(player_id, tournament_id):
    tournament = get_tournament(tournament_id)
    player = get_player(player_id)
    
    # Base eligibility
    if tournament.qualification_required:
        required_elo = tournament.min_elo
        if player.current_elo < required_elo:
            return False
    
    # League eligibility
    if tournament.league_restricted:
        if player.league_tier not in tournament.allowed_leagues:
            return False
    
    # Reputation eligibility
    if tournament.requires_verified:
        if not player.kyc_verified:
            return False
    
    # Trust score check
    if tournament.min_trust_score:
        if player.trust_score < tournament.min_trust_score:
            return False
    
    return True
```

### Tournament Bracket Structure:

```
Single Elimination (Daily/Weekly):
  Round 1: 256 → 128 matches
  Round 2: 128 → 64 matches
  Round 3: 64 → 32 matches
  Round 4: 32 → 16 matches (Quarterfinals)
  Round 5: 16 → 8 matches (Semifinals)
  Round 6: 8 → 4 matches (Semifinals)
  Round 7: 4 → 2 matches (Finals)
  Round 8: 2 → 1 match (Champion)

Prize Distribution (Example Weekly):
  1st place (champion): 75 tokens
  2nd place: 40 tokens
  3rd-4th place: 20 tokens each
  5th-8th place: 10 tokens each
  9th-32nd place: 5 tokens each
```

### Prize Pool Calculation:

```
Tournament_Entry_Fee = 25 tokens
Max_Participants = 256

Total_Collected = 256 × 25 = 6400 tokens
Platform_Fee = 10% × 6400 = 640 tokens
Prize_Pool = 6400 - 640 = 5760 tokens

Distribution:
  Winners fund = 5760 tokens
  Platform revenue = 640 tokens (to Platform Revenue Reserve)
```

### Business Logic:
- **All tournaments treasury-backed** (prizes funded before tournament starts)
- **No overpayment:** Scale prizes down if fewer participants
- **Guaranteed payouts:** Winners always receive promised rewards
- **Escalating stakes:** Higher tournaments = higher entry fees and prizes
- **Prestigious display:** Tournament wins displayed prominently on profile

---

## 8. TREASURY & FINANCIAL SYSTEM

### Core APIs Needed:
```
GET  /treasury/balance          → Get reserve balances
GET  /treasury/liabilities      → Get total liabilities
GET  /treasury/coverage-ratio   → Calculate solvency
GET  /treasury/audit-report     → Get financial audit
POST /treasury/reconcile        → Trigger reconciliation
GET  /treasury/health-score     → Get treasury health
```

### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `treasury_accounts` | Reserve accounts | account_id, account_type, balance, min_threshold, status |
| `reserves` | Segregated funds | reserve_id, name, balance, purpose, monthly_target, max_balance |
| `financial_ledger` | Double-entry accounting | entry_id, timestamp, debit_account, credit_account, amount, description, verified_by |
| `reserve_snapshots` | Daily backup | snapshot_id, snapshot_date, reserve_balances_json, calculated_liabilities |

### Reserve Structure (Segregated Funds):

**1. Player Funds Reserve**
- **Purpose:** Money belonging to players
- **Calculation:** Sum of all player wallet balances
- **Rules:** 
  - MUST always equal sum of all player wallets
  - Cannot be used for operations/marketing/development
  - Highest priority claim on platform assets
  - Immobilized (cannot be touched)
- **Target:** 100% coverage ratio (Liabilities ≤ Reserve)

**2. Platform Revenue Reserve**
- **Sources:** 
  - PvP platform fees (10% of match pots)
  - Tournament platform fees
  - Premium season pass sales
  - Cosmetic sales (future)
  - Sponsorships (future)
- **Uses:**
  - Operations (server costs, salaries)
  - Development (new features)
  - Marketing (user acquisition)
- **Strategy:** Reinvest 80%, allocate 20% to other reserves

**3. Season Fund Reserve**
- **Purpose:** Season reward payouts
- **Funding:** 5% of Platform Revenue
- **Cycle:** Replenish monthly to ensure sufficient for next season
- **Controls:** 
  - Threshold alert if < 1 month funding
  - Manual approval for large disbursements

**4. Championship Fund Reserve**
- **Purpose:** Tournament prize pools
- **Funding:** 10% of Platform Revenue
- **Calculation:** 
  - Calculate monthly tournament volume
  - Reserve 3x monthly amount
- **Release:** Locked until tournament begins

**5. Jackpot Fund Reserve**
- **Purpose:** Special promotion prizes, bonus pools
- **Funding:** 5% of Platform Revenue
- **Uses:** Limited-time high-reward challenges
- **Max Payout:** Single event capped at fund size ÷ 10

**6. Emergency Reserve**
- **Purpose:** Contingency buffer for crises
- **Target:** 20% of Platform Revenue Reserve
- **Uses:** System failures, fraud recovery, regulatory fines
- **Release:** CEO + CFO approval required

### Treasury Architecture:

```
Financial Flow:

INFLOW:
  Player Deposits → Payment Provider → Confirmed → 
  Player Funds Reserve + Platform Revenue
  
MATCH ENTRY:
  Entry Fees → Locked Wallet → Match Completes →
  Winner's Prize from Prize Pool → Ledger Entry

REWARDS:
  Season/Tournament Rewards → Championship Fund →
  Ledger Entry → Player Wallet (LOCKED until withdrawal)
  
WITHDRAWAL:
  Player Initiates → KYC Check → AML Screen →
  Withdrawal Fee (1%) → Payment Provider →
  Bank Transfer → Player Funds decreased
```

### Financial Philosophy:

**Core Rules:**
1. **Auditability:** Every token tracked
2. **Reconciliation:** Ledger must match reality daily
3. **Coverage:** Liabilities never exceed reserves
4. **Segregation:** Funds never mixed without approval
5. **Priority:** Player funds > all other claims
6. **Immutability:** Financial records never deleted

### Solvency Calculations:

```
Total_Player_Liabilities = Sum(all_player_wallet_balances)
Player_Funds_Reserve_Balance = ?
Coverage_Ratio = Player_Funds_Reserve / Total_Player_Liabilities

Target_Ratio = 1.0 (100% coverage)
Alert_Yellow = 0.95 (95% - warning)
Alert_Red = 0.90 (90% - emergency)

Solvency_Status:
  IF Coverage_Ratio >= 1.0: GREEN (Fully solvent)
  IF 0.95 <= Coverage_Ratio < 1.0: YELLOW (Minor deficit)
  IF Coverage_Ratio < 0.95: RED (Major concern)

Emergency Actions (if RED):
  1. Halt all withdrawals > USD $1000
  2. Notify CEO
  3. Audit immediately
  4. Consider freezing platform
  5. Legal/regulatory notification
```

### Ledger Structure (Double-Entry):

```
Every transaction has two entries (debit ≠ credit):

Example 1: Player deposits USD $100 (10 tokens)
  Entry 1: DEBIT Player Funds Reserve (10 tokens)
  Entry 2: CREDIT Bank Account (USD 100 + 2.5% fee)
  
Example 2: Player wins 20-token PvP match
  Entry 1: DEBIT Prize Pool Account (20 tokens)
  Entry 2: CREDIT Player Wallet (20 tokens)
  
Example 3: Platform collects 2-token fee (10% of 20-token pot)
  Entry 1: DEBIT PvP Match Pot (2 tokens)
  Entry 2: CREDIT Platform Revenue Reserve (2 tokens)

All entries immutable once recorded.
```

### Business Logic:
- **No Negative Balances:** All accounts start at 0, can't go negative
- **Atomic Transactions:** All-or-nothing, no partial transactions
- **Delayed Settlement:** Some transactions settle after 24 hours (fraud check)
- **Monthly Rebalancing:** Move funds between reserves to maintain targets

---

## 9. ANTI-CHEAT & SECURITY SYSTEM

### Core APIs Needed:
```
POST /security/validate-move     → Validate game move
GET  /security/risk-score        → Get account risk
POST /security/device-check      → Verify device
GET  /security/audit-log         → Get security events
POST /security/report-exploit    → Report suspicious activity
```

### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `device_fingerprints` | Device tracking | fingerprint_id, user_id, device_hash, os, browser, last_seen, first_seen |
| `anti_bot_scores` | Bot detection | score_id, user_id, risk_score, last_updated, signals_detected |
| `security_events` | Incident log | event_id, user_id, event_type, severity, timestamp, details |
| `replay_verification` | Replay validation | replay_id, verification_hash, status, verified_date |
| `suspicious_accounts` | Flagged accounts | account_id, user_id, reason, status, investigation_date |

### Server Authoritative Architecture:

**Client Trust Level = ZERO for:**
- Wallet balance calculations
- XP and reward generation
- Ranking calculations
- Match result determination
- Challenge seed generation
- All business logic

**Client Trusted Only For:**
- Display rendering
- Audio playback
- Visual effects
- User input (coordinates, clicks)

**Server Must Always:**
- Validate every move
- Recalculate balances
- Verify rankings
- Regenerate challenges
- Confirm all results

### Movement Validation:

```python
def validate_move(match_id, player_id, move_data):
    """
    Validates every player move server-side.
    CLIENT MOVE NEVER TRUSTED.
    """
    match = get_match(match_id)
    challenge = get_challenge(match.maze_id)
    
    # Check move is valid in maze
    current_position = get_player_position(match_id, player_id)
    new_x, new_y = move_data['coordinates']
    
    # 1. Is new position in valid path?
    if not is_valid_path(challenge, current_position, (new_x, new_y)):
        return False, "Invalid move - not in maze path"
    
    # 2. Is player alive?
    if not is_player_alive(match_id, player_id):
        return False, "Player already lost"
    
    # 3. Is move within game rules?
    if not follows_game_rules(match_id, move_data):
        return False, "Violates game rules"
    
    # 4. Is timing valid?
    if move_time_invalid(match, move_data):
        return False, "Move timing invalid"
    
    # If all checks pass, accept move
    update_player_position(match_id, player_id, (new_x, new_y))
    log_move(match_id, player_id, move_data, timestamp=now())
    return True, "Move accepted"
```

### Anti-Bot Engine:

**Bot Detection Signals:**
- **Reaction Time Consistency:** Humans vary; bots are consistent
  - Humans: 150-500ms reaction time, variable
  - Bots: 10-50ms, highly consistent
  - Alert if: < 100ms for 100+ moves
  
- **Movement Precision:** Humans make mistakes; bots don't
  - Optimal path efficiency > 95%: Suspicious
  - Alert if: > 10 consecutive perfect moves
  
- **Completion Consistency:** Humans perform variably
  - Same maze completion time ± 2%: Suspicious
  - Alert if: 50+ identical completion times
  
- **Input Patterns:** Bots follow patterns
  - Click timing regularity
  - Movement direction patterns
  - Decision point analysis

**Risk Scoring:**

```
Risk_Score = 0-100

Calculation:
  reaction_time_score = analyze_reaction_times()      # 0-25
  movement_precision_score = analyze_precision()      # 0-25
  completion_consistency_score = analyze_consistency()  # 0-25
  input_pattern_score = analyze_patterns()            # 0-25
  
  Risk_Score = sum(all_scores)
  
Green (0-30): Normal human behavior
Yellow (31-70): Suspicious patterns detected
Red (71-100): High probability of automation

Actions:
  Green: Allow normal play
  Yellow: Monitor closely, require verification challenge
  Red: Suspend account, flag for investigation
```

### Device Fingerprinting:

```python
def generate_fingerprint(request):
    """
    Creates unique device identifier from hardware characteristics
    """
    fingerprint_data = {
        'user_agent': request.headers['User-Agent'],
        'screen_resolution': request.body['screen_resolution'],
        'timezone': request.body['timezone'],
        'language': request.body['language'],
        'hardware_concurrency': request.body['cpu_cores'],
        'device_memory': request.body['memory_gb'],
        'gpu_info': request.body['gpu_model'],
        'installed_plugins': request.body['browser_plugins'],
        'canvas_fingerprint': request.body['canvas_hash'],
        'webgl_data': request.body['webgl_vendor'],
    }
    
    device_hash = SHA256(json.dumps(fingerprint_data))
    return device_hash

def check_account_sharing(user_id, new_fingerprint):
    """
    Detect if account is being used from multiple devices
    """
    known_devices = get_user_devices(user_id)
    
    if new_fingerprint in known_devices:
        return "Known device"
    
    # Check if too many new devices in short time
    new_devices_30d = get_new_devices(user_id, days=30)
    if len(new_devices_30d) > 3:
        return "High device churn - possible account sharing"
    
    # Check if devices in different geographic locations simultaneously
    if impossible_travel_detected(user_id):
        return "Impossible travel - simultaneous locations"
    
    return "Device added to whitelist"
```

### Replay Verification:

```python
def verify_replay(replay_id):
    """
    Cryptographically verify replay authenticity
    """
    replay = get_replay(replay_id)
    match = get_match(replay.match_id)
    
    # 1. Recreate challenge from seed
    expected_challenge = generate_challenge(
        maze_seed=match.maze_seed,
        difficulty=match.difficulty_score
    )
    
    # 2. Replay all moves
    for frame in replay.frames:
        # Validate frame move against challenge
        if not validate_move_frame(expected_challenge, frame):
            return False, "Frame contains invalid move"
    
    # 3. Verify hash
    expected_hash = SHA256(serialize_replay_frames(replay.frames))
    if expected_hash != replay.verification_hash:
        return False, "Replay hash mismatch - tampered"
    
    # 4. Check completion time
    if replay.completion_time != match.completion_time:
        return False, "Time mismatch"
    
    return True, "Replay verified authentic"
```

### Security Rules:

**Authentication:**
- Email + Password (12 chars minimum, complex)
- MFA required for withdrawals
- Device fingerprinting
- Session timeout: 30 days

**API Security:**
- JWT token validation on every request
- Rate limiting: 1000 requests/minute per user
- Input sanitization on all fields
- SQL injection prevention (parameterized queries)
- XSS prevention (output encoding)
- CSRF token on state-changing requests

**Infrastructure Security:**
- TLS 1.3+ for all communications
- Encrypted database at rest
- Encrypted backups
- VPN/firewall for internal services
- Zero-trust network architecture

---

## 10. REPLAY & VERIFICATION SYSTEM

### Core APIs Needed:
```
GET  /replays/{replay_id}       → Get replay data
POST /replays/{replay_id}/verify → Verify replay authenticity
GET  /replays/history           → Get user replay list
POST /replays/{replay_id}/report → Report replay issue
```

### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `replays` | Replay records | replay_id, match_id, recording_data, verification_hash, created_at, expires_at |
| `replay_frames` | Movement data | frame_id, replay_id, frame_number, position_x, position_y, move_type, timestamp |
| `replay_metadata` | Replay info | replay_id, duration_seconds, player_1_name, player_2_name, final_result, winner_id |
| `replay_disputes` | Challenges | dispute_id, replay_id, complainant_id, reason, status, resolution |

### Replay Recording:

**What Gets Recorded:**
- Player 1 position every frame
- Player 2 position every frame
- Lives remaining (both players)
- Completion percentage
- Timer state
- Events (traps, deaths, completion)
- Final result and timing

**Storage:**
- Compressed movement data (not video)
- Deterministic: Same input always generates same output
- Size: ~10KB per match typically
- Retention: 90 days minimum, archivable after

**Compression:**
- Frame-to-frame delta encoding (only position changes)
- Timestamps encoded efficiently
- No video file (just player coordinates)
- Playable back from seed + frame data

### Verification Features:

**Deterministic Replay:**
- Start with maze seed
- Feed recorded moves
- Regenerate exact same challenge state
- Verify final result matches

**Cryptographic Verification:**
- Replay hash created at completion
- Hash includes all move data
- Tampering detected immediately
- Certificate chain verifies server authority

**Dispute Resolution:**
- Player challenges match outcome
- Admin can replay and verify
- Evidence stored permanently
- Reversal possible only if fraud detected

### Replay Use Cases:

1. **Player Verification:** "Did I actually lose this match?"
2. **Spectator Viewing:** Tournament spectators watch live replays
3. **Leaderboard Verification:** Verify top scores are legitimate
4. **Cheat Detection:** Identify impossible patterns
5. **Dispute Resolution:** Settle player complaints
6. **Platform Analytics:** Analyze gameplay patterns

---

## 11. LEADERBOARDS & RANKINGS

### Core APIs Needed:
```
GET  /leaderboards/global     → Top 1000 globally
GET  /leaderboards/country/{code} → Country rankings
GET  /leaderboards/league/{tier}  → League-specific
GET  /leaderboards/season/{id}    → Seasonal standings
GET  /player/{id}/rank        → Get specific player rank
```

### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `leaderboard_snapshots` | Cached rankings | snapshot_id, snapshot_date, board_type, rank_data_json |
| `player_rankings` | Current position | user_id, global_rank, country_rank, league_rank, elo_rating, last_updated |
| `historic_rankings` | Historical tracking | ranking_id, user_id, date, rank_position, elo_rating |

### Leaderboard Types:

| Type | Frequency | Criteria | Visibility |
|------|-----------|----------|-----------|
| Global | Real-time | ELO rating | Top 1000 public |
| Country | Daily | ELO + Country | Top 100 per country |
| League | Daily | League tier + points | All players in league |
| Seasonal | Daily | Season points | Reset each season |
| Tournament | Real-time | Tournament placement | During tournament |

### Ranking Calculation:

```python
def calculate_global_rankings():
    """
    Update global leaderboard daily
    """
    players = get_all_active_players()
    
    rankings = []
    for player in players:
        elo = get_current_elo(player.id)
        country = player.country
        
        rankings.append({
            'user_id': player.id,
            'elo_rating': elo,
            'country': country,
            'matches_played': get_match_count(player.id),
            'win_rate': calculate_win_rate(player.id),
        })
    
    # Sort by ELO descending
    rankings.sort(key=lambda x: x['elo_rating'], reverse=True)
    
    # Assign ranks
    for rank, player_data in enumerate(rankings, start=1):
        player_data['global_rank'] = rank
        
        if rank <= 1000:  # Only top 1000 shown
            player_data['public_visible'] = True
    
    # Store snapshot
    store_leaderboard_snapshot(rankings)
    
    return rankings
```

### Private Profile Data:

- **Shown on Profile:** ELO, rank, wins, losses, win rate
- **Hidden from Public:** Exact withdrawal times, payment methods, full transaction history
- **Admin Only:** Risk scores, KYC status, security flags

---

## 12. ACHIEVEMENTS & LEGACY SYSTEM

### Core APIs Needed:
```
GET  /achievements             → Get all achievements
GET  /achievements/{id}        → Get specific achievement
GET  /player/achievements      → Get player achievements
GET  /player/legacy            → Get legacy info
POST /achievements/{id}/claim  → Claim reward
```

### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `achievements` | Achievement definitions | achievement_id, name, description, requirement_type, reward_xp, icon_url |
| `user_achievements` | Player progress | user_achievement_id, user_id, achievement_id, earned_date, progress_percent |
| `legacy_ranks` | Legacy progression | user_id, legacy_points, rank_tier, achievements_earned, hall_of_fame |

### Achievement Categories:

**Gameplay Achievements:**
- Win 10 PvP matches
- Win 25 PvP matches
- Achieve 50-match win streak
- Complete all house tiers
- Win tournament

**Progression Achievements:**
- Reach XP level 100
- Reach Prestige I
- Reach Diamond league
- Earn 1000 legacy points
- Complete seasonal challenge

**Social Achievements:**
- Invite 5 friends
- Create clan
- Participate in clan tournament
- Reach 100 followers

**Special Achievements:**
- Founder status (before launch)
- Early adopter (first 10k players)
- Platform ambassador
- Bug bounty contributor

### Legacy System:

```
Legacy Tiers (Based on Legacy Points):
  Bronze: 0-100 points
  Silver: 101-500 points
  Gold: 501-1500 points
  Platinum: 1501-3000 points
  Diamond: 3001-5000 points
  Legend: 5001+ points

Hall of Fame:
  Top 1000 legacy point holders
  Lifetime recognition
  Special badge on profile
  Featured on platform

Legacy Points Earned From:
  Season participation: +10 points
  Tournament success: +50 points (per placement)
  House completion: +5 points (per tier)
  PvP activity: +1 point (per match)
  Seasonal achievement: +100 points
  Founder status: +500 points (one-time)
```

---

# PHASE 3: FINANCIAL INFRASTRUCTURE & COMPLIANCE

## 3. CORE COMPONENTS

### 1. ADVANCED TREASURY & DOUBLE-ENTRY LEDGER

### Core APIs Needed:
```
GET  /treasury/accounts         → Get all reserve accounts
GET  /treasury/balance          → Get current balance
POST /treasury/reconcile        → Start reconciliation
GET  /treasury/reconcile-status → Check reconciliation progress
GET  /treasury/audit-report     → Generate financial audit
GET  /treasury/solvency         → Calculate solvency ratio
```

### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `treasury_accounts` | Reserve accounts | account_id, account_name, account_type, balance, min_threshold, max_capacity, status |
| `double_entry_ledger` | All transactions | entry_id, transaction_id, debit_account_id, credit_account_id, amount, description, approved_by, timestamp |
| `reserve_snapshots` | Daily backups | snapshot_id, snapshot_date, all_reserves_json, total_liabilities, coverage_ratio |
| `reconciliation_logs` | Audit trail | log_id, reconciliation_date, discrepancies_found, resolution, status |

### Financial Double-Entry Pattern:

```
Every transaction creates TWO immutable ledger entries:
  Entry 1: DEBIT (fund decreases)
  Entry 2: CREDIT (fund increases)
  Sum(Debits) = Sum(Credits) → Accounting Equation Always Balanced

Example: Player deposits USD $100
  DEBIT:  Bank Account                +100 USD
  CREDIT: Player Funds Reserve        -100 USD
  (converted to tokens at 1:10 rate)

Example: Player wins 20-token PvP
  DEBIT:  Match Prize Pool Account    +20 tokens
  CREDIT: Player Wallet               -20 tokens

All entries:
  - Immutable (never changed after creation)
  - Timestamped (UTC with milliseconds)
  - Approved (at least one signature)
  - Auditable (full chain preserved)
```

### Reserve Reconciliation Workflow:

```
1. COLLECTION PHASE
   - Fetch all player wallet balances
   - Sum = Expected Player Funds Reserve
   - Fetch all reserve account balances
   - Get external payment provider balances
   - Get bank account balance

2. VALIDATION PHASE
   - Player Funds Reserve ≥ Sum(Player Wallets)?
   - All reserve accounts positive?
   - No orphaned transactions?
   - All ledger entries balanced?

3. COMPARISON PHASE
   - Internal records vs external statements
   - Payment provider confirmations
   - Bank statement matching
   - Discrepancy analysis

4. ADJUSTMENT PHASE
   - Identify root cause of any discrepancies
   - Create adjustment journal entries if needed
   - Document all changes
   - Obtain approval signatures

5. REPORTING PHASE
   - Generate reconciliation report
   - Archive for audit
   - Alert if issues found
   - Update executive dashboard

Frequency:
  Real-time: Critical accounts (player funds, bank)
  Daily: Comprehensive reconciliation
  Weekly: Deep financial analysis
  Monthly: Executive reporting
```

### Solvency Monitoring:

```python
def calculate_solvency_ratio():
    """
    Track platform ability to cover all player liabilities
    """
    total_player_wallets = sum_all_player_balances()
    player_funds_reserve = get_reserve_balance('PLAYER_FUNDS')
    
    # Coverage ratio = how many times over can reserves cover liabilities
    coverage_ratio = player_funds_reserve / total_player_wallets
    
    # Treasury health score (0-100)
    if coverage_ratio >= 1.10:
        health_score = 100
        status = "EXCELLENT"
    elif coverage_ratio >= 1.05:
        health_score = 90
        status = "GOOD"
    elif coverage_ratio >= 1.0:
        health_score = 75
        status = "ADEQUATE"
    elif coverage_ratio >= 0.98:
        health_score = 50
        status = "CONCERNING"
    else:
        health_score = 0
        status = "CRITICAL"
    
    # Alert thresholds
    if coverage_ratio < 0.95:
        send_alert_to_cfo("SOLVENCY_RED", coverage_ratio)
    elif coverage_ratio < 1.0:
        send_alert_to_cfo("SOLVENCY_YELLOW", coverage_ratio)
    
    return {
        'coverage_ratio': coverage_ratio,
        'health_score': health_score,
        'status': status,
        'total_liabilities': total_player_wallets,
        'available_reserves': player_funds_reserve,
    }
```

---

### 2. PAYMENT PROVIDER INTEGRATION

### Core APIs Needed:
```
POST /payments/deposit          → Initiate deposit
POST /payments/withdrawal       → Initiate withdrawal
GET  /payments/status/{id}      → Check payment status
GET  /payments/methods          → Get available payment methods
POST /payments/webhook          → Receive provider updates
GET  /payments/reconcile        → Reconcile with provider
```

### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `payment_providers` | Provider configs | provider_id, provider_name, api_endpoint, status, supported_regions |
| `deposits` | Deposit records | deposit_id, user_id, amount_local_currency, amount_tokens, provider, status, provider_tx_id |
| `withdrawals` | Withdrawal records | withdrawal_id, user_id, amount_tokens, amount_local_currency, provider, bank_account_id, status |
| `payment_audit` | Transaction log | audit_id, payment_id, timestamp, old_status, new_status, provider_response |

### Deposit Processing Flow:

```
1. USER INITIATES
   - Select payment method
   - Enter amount (min USD $10)
   - Enter payment details
   - Submit

2. SERVER PROCESSES
   - Validate amount > minimum
   - Check payment method available in region
   - Create deposit record (status: PENDING)
   - Generate payment session with provider
   - Redirect user to provider

3. PAYMENT PROVIDER PROCESSES
   - Collect payment
   - Perform fraud checks
   - Confirm or decline

4. WEBHOOK CONFIRMATION
   - Provider sends webhook: CONFIRMED
   - Server validates webhook signature
   - Create ledger entries:
     * DEBIT: Bank Account
     * CREDIT: Player Funds Reserve
   - Credit player wallet (status: COMPLETED)
   - Emit notification: "Deposit Received"

5. PLAYER SEES
   - Balance updated immediately
   - Can use tokens for matches
   - Transaction in history
```

### Withdrawal Processing Flow:

```
1. USER INITIATES
   - Enter withdrawal amount
   - Confirm bank account
   - Submit

2. VERIFICATION PHASE
   - Check account verified (KYC)
   - Check wallet balance ≥ amount
   - Check within daily/monthly limits
   - Lock tokens (debit Live, credit Pending)
   - Status: VERIFICATION_PENDING

3. AML SCREENING
   - Screen against OFAC list
   - Check for suspicious patterns
   - Review transaction history
   - If suspicious: Flag for manual review
   - Status: AML_SCREENING_COMPLETE

4. ADMIN REVIEW (if high value or flagged)
   - Manual verification
   - Document review
   - Approve or deny
   - If approved: Status: APPROVED

5. PAYMENT PROVIDER INITIATION
   - Create withdrawal instruction
   - Send to bank/payment provider
   - Track provider transaction ID
   - Status: PROCESSING

6. SETTLEMENT
   - 1-3 business days
   - Provider confirms
   - Create ledger entries:
     * DEBIT: Player Funds Reserve
     * CREDIT: Bank Account
   - Unlock tokens
   - Player notified: "Withdrawal Complete"
   - Status: COMPLETED

Possible outcomes:
  - APPROVED → Processing → COMPLETED
  - DENIED → Tokens unlocked, returned to wallet
  - FAILED → Provider error, retry or refund
```

### Supported Payment Methods:

| Provider | Regions | Processing Time | Fees |
|----------|---------|-----------------|------|
| Stripe (Cards) | Global | Instant | 2.9% + $0.30 |
| EFT | South Africa | 1-2 days | 2% |
| Instant EFT | South Africa | Minutes | 3% |
| PayFast | South Africa, Africa | Minutes | 2.5% |
| Ozow | South Africa | Minutes | 1% |

---

### 3. AML & COMPLIANCE SYSTEM

### Core APIs Needed:
```
POST /compliance/aml-screen     → Screen transaction for AML
GET  /compliance/kyc-status     → Get KYC verification status
POST /compliance/kyc-submit     → Submit KYC documents
GET  /compliance/risk-score     → Get account risk level
GET  /compliance/report         → Generate compliance report
```

### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `aml_screenings` | AML checks | screening_id, user_id, amount, direction (deposit/withdrawal), status, provider_response, screening_date |
| `kyc_records` | KYC verification | kyc_id, user_id, verification_status, document_type, verification_date, verified_by, expiration_date |
| `compliance_events` | Compliance incidents | event_id, user_id, event_type, severity, timestamp, description, resolved |

### KYC (Know Your Customer) Process:

```
Tier 1: Basic Registration (No KYC)
  - Create account
  - Email verification
  - Play with demo wallet
  - Limit: Can't deposit or withdraw

Tier 2: Email Verified
  - Verify email
  - Can deposit up to USD $1000/month
  - Can withdraw up to USD $500/month
  - Auto-screened deposits

Tier 3: Basic KYC (Identity Verified)
  - Submit government ID
  - Manual verification by provider
  - Can deposit USD $50,000/month
  - Can withdraw USD $10,000/month
  - Enhanced AML screening

Tier 4: Enhanced KYC (Full Verification)
  - Submit proof of address
  - Source of funds verification
  - Manual senior review
  - Can deposit/withdraw unlimited
  - Real-time monitoring

Enhanced Verification Triggers:
  - Withdrawal > USD $50,000
  - Deposit > USD $100,000
  - Rapid deposit/withdrawal pattern
  - Flagged for suspicious activity
```

### AML Screening Rules:

```python
def aml_screen_transaction(user_id, amount, direction):
    """
    Screen transaction against AML rules
    """
    user = get_user(user_id)
    
    # Check OFAC list (US sanctions)
    if is_ofac_listed(user.name, user.country):
        return BLOCKED, "OFAC listing detected"
    
    # Check transaction velocity
    recent_transactions = get_transactions_24h(user_id)
    if sum(recent_transactions) > threshold_for_tier(user.kyc_tier):
        return FLAGGED, "Velocity threshold exceeded"
    
    # Check unusual patterns
    if is_unusual_pattern(user_id, amount, direction):
        return FLAGGED, "Unusual transaction pattern"
    
    # Check if account is young with large transaction
    account_age_days = (now() - user.created_at).days
    if account_age_days < 30 and amount > 1000:
        return FLAGGED, "Large transaction on young account"
    
    # Check for structuring (rapid small transactions)
    if detect_structuring(user_id):
        return FLAGGED, "Possible structuring detected"
    
    # If all checks pass
    return ALLOWED, "Transaction approved"
```

### Compliance Reporting:

```
Daily Compliance Report:
  - Total deposits processed
  - Total withdrawals processed
  - Flagged transactions count
  - Blocked transactions count
  - KYC verifications completed
  - Suspicious activity investigations

Monthly Regulatory Filing:
  - Transaction report to authorities (if required)
  - Sanctions list hits
  - Customer due diligence updates
  - Incident summary
```

---

### 4. FRAUD DETECTION & RISK MANAGEMENT

### Core APIs Needed:
```
GET  /fraud/risk-score/{user_id}  → Get account risk
POST /fraud/report-suspicious      → Report suspicious activity
GET  /fraud/cases                  → List fraud investigations
POST /fraud/cases/{id}/resolve     → Resolve case
GET  /disputes/open               → List open disputes
```

### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `risk_scores` | Player risk assessment | risk_id, user_id, financial_risk, behavioral_risk, account_risk, overall_score, updated_at |
| `fraud_cases` | Investigation records | case_id, case_type, user_id, amount, status, created_at, resolved_at, resolution |
| `dispute_records` | Complaint tracking | dispute_id, complainant_id, respondent_id, reason, evidence, status, ruling |

### Risk Categories:

**Financial Fraud:**
- Chargebacks on deposits
- Stolen payment methods
- Money laundering patterns
- Unusual fund flows

**Account Abuse:**
- Account sharing
- Multiple accounts per player
- Credential sharing
- Unauthorized access

**Collusion:**
- Players coordinating wins
- Intentional losses
- Match fixing
- Reward sharing schemes

**Match Manipulation:**
- Obvious throws
- Impossible performance
- Bot usage
- Exploit abuse

**Bonus Abuse:**
- Sign-up bonus grinding
- Fake accounts for bonuses
- Welcome offer fraud
- Referral exploitation

**Payment Abuse:**
- Deposit and rapid withdrawal
- Chargeback after winning
- Multiple failed attempts
- Card testing

### Dynamic Risk Scoring:

```python
def calculate_risk_score(user_id):
    """
    Continuously evaluate account risk
    Scale: 0-100 (higher = riskier)
    """
    user = get_user(user_id)
    
    # Financial Risk (0-25)
    financial_score = 0
    deposits_last_30d = get_deposits_total(user_id, days=30)
    withdrawals_last_30d = get_withdrawals_total(user_id, days=30)
    
    if deposits_last_30d == 0:
        financial_score += 10  # No deposits yet
    elif deposits_last_30d > 50000:
        financial_score += 15  # Unusually high
    
    if withdrawals_last_30d / deposits_last_30d > 0.8:
        financial_score += 10  # Rapid withdrawal pattern
    
    chargebacks = count_chargebacks(user_id)
    financial_score += min(chargebacks * 5, 25)
    
    # Behavioral Risk (0-25)
    behavioral_score = 0
    
    win_rate = get_win_rate(user_id)
    if win_rate > 0.95:
        behavioral_score += 15  # Suspiciously high
    
    # Check for bot behavior
    bot_signals = count_bot_signals(user_id)
    behavioral_score += min(bot_signals * 3, 25)
    
    # Account Risk (0-25)
    account_score = 0
    
    if user.kyc_status == "UNVERIFIED":
        account_score += 10
    
    device_count = count_unique_devices(user_id)
    if device_count > 5:
        account_score += 15  # Multiple devices
    
    location_diversity = check_location_diversity(user_id)
    if location_diversity > 3:
        account_score += 10  # Impossible travel
    
    # Overall Score
    overall_score = financial_score + behavioral_score + account_score
    
    # Risk Level
    if overall_score < 20:
        risk_level = "GREEN"
    elif overall_score < 50:
        risk_level = "YELLOW"
    else:
        risk_level = "RED"
    
    return {
        'overall_score': overall_score,
        'risk_level': risk_level,
        'financial_score': financial_score,
        'behavioral_score': behavioral_score,
        'account_score': account_score,
    }
```

### Dispute Resolution Workflow:

```
1. COMPLAINT FILED
   - Player submits dispute
   - Provide evidence/reason
   - Case created (status: OPEN)

2. REVIEW PHASE
   - Examine replay data
   - Check match logs
   - Verify server calculations
   - Review player history

3. INVESTIGATION
   - If fraud suspected: Deeper investigation
   - Interview if needed
   - External evidence gathering
   - Policy application

4. DECISION
   - Uphold original result OR
   - Reverse (refund entry fee) OR
   - Compensation (partial refund)

5. COMMUNICATION
   - Notify both parties
   - Document reasoning
   - Update case status: RESOLVED

6. APPEAL
   - Option to appeal within 7 days
   - Different reviewer
   - Final decision final

Case Statuses:
  OPEN → UNDER_REVIEW → DECIDED → RESOLVED
  At any point can be ESCALATED
```

---

### 5. RECONCILIATION & SOLVENCY ENGINE

### Core APIs Needed:
```
POST /reconciliation/start       → Start reconciliation process
GET  /reconciliation/status      → Check progress
GET  /reconciliation/report      → Get results
POST /reconciliation/adjust      → Create adjustment entry
GET  /solvency/metrics           → Get solvency status
```

### Reconciliation Workflow:

```
Step 1: BALANCE COLLECTION (Automated, Real-time)
  - Fetch all player wallet balances → Sum
  - Fetch all reserve account balances
  - Fetch payment provider balances
  - Fetch bank statement (if available)
  - Fetch locked tokens in matches

Step 2: INTERNAL VALIDATION (Automated, Daily)
  - Player wallets ≥ 0 all
  - No negative balances
  - Sum of wallet types = total
  - No orphaned transactions

Step 3: LEDGER VERIFICATION (Automated, Daily)
  - Verify accounting equation: Debits = Credits
  - Check no duplicate entries
  - Verify all amounts non-negative
  - Check timestamps in order

Step 4: PROVIDER RECONCILIATION (Automated, Daily)
  - Compare internal deposit records vs provider
  - Match transaction IDs
  - Verify amounts
  - Identify discrepancies

Step 5: SOLVENCY CHECK (Automated, Real-time)
  - Calculate coverage ratio
  - Compare to minimums
  - Alert if concerning

Step 6: ADJUSTMENT PHASE (Manual, if needed)
  - Identify discrepancies requiring adjustment
  - Create journal entries to correct
  - Document reasons
  - Require approval signatures

Step 7: REPORTING (Automated, Daily)
  - Generate reconciliation report
  - Archive for audit
  - Alert executives if issues
  - Update dashboards
```

### Treasury Health Scoring:

```
Coverage Ratio = Total Reserves / Total Liabilities

Health Score Calculation:
  IF Coverage_Ratio >= 1.20: Health_Score = 100 (EXCELLENT)
  IF Coverage_Ratio >= 1.10: Health_Score = 90 (VERY GOOD)
  IF Coverage_Ratio >= 1.05: Health_Score = 80 (GOOD)
  IF Coverage_Ratio >= 1.00: Health_Score = 70 (ADEQUATE)
  IF Coverage_Ratio >= 0.98: Health_Score = 50 (CONCERNING)
  IF Coverage_Ratio >= 0.95: Health_Score = 30 (WARNING)
  IF Coverage_Ratio < 0.95: Health_Score = 0 (CRITICAL)

Alert Levels:
  Coverage >= 1.0: GREEN (All liabilities covered)
  0.95 <= Coverage < 1.0: YELLOW (Minor concern, review plan)
  Coverage < 0.95: RED (Emergency, halt withdrawals, notify CEO)
```

---

### 6. REPORTING & ANALYTICS INFRASTRUCTURE

### Core APIs Needed:
```
GET  /reports/executive         → Get executive summary
GET  /reports/treasury          → Get treasury report
GET  /reports/compliance        → Get compliance report
GET  /reports/risk              → Get risk report
GET  /dashboards/treasury       → Real-time treasury dashboard
GET  /dashboards/executive      → Real-time executive dashboard
```

### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `report_definitions` | Report templates | report_id, report_name, frequency, recipients, template_json |
| `generated_reports` | Report instances | generated_id, report_id, generation_date, data_json, status |
| `dashboard_metrics` | Cached metrics | metric_id, metric_type, value, calculated_at |

### Executive Dashboard (Real-Time):

**Key Metrics:**
- Active Players (today, week, month)
- New Players (today, week, month)
- Deposits (USD amount, token amount)
- Withdrawals (USD amount, token amount)
- Platform Revenue (fees collected)
- Treasury Health Score (0-100)
- Reserve Coverage Ratio
- Solvency Status (Green/Yellow/Red)
- Fraud Alert Count
- System Uptime %

**Visualizations:**
- Revenue trend (30-day chart)
- Player growth curve
- Deposit vs Withdrawal comparison
- Reserve balance trend
- Active match volume
- Regional breakdown

---

### 7. DATABASE SCHEMA OVERVIEW (Phase 3)

**Primary Domains:**

| Domain | Purpose | Tables |
|--------|---------|--------|
| Users | Identity & profile | users, user_profiles, user_sessions, devices |
| Wallets | Token management | wallets, transactions, ledger_entries |
| Treasury | Financial reserves | treasury_accounts, reserves, reserve_snapshots |
| Transactions | All payments | deposits, withdrawals, settlement_records |
| Compliance | KYC & AML | kyc_records, aml_screenings, compliance_events |
| Fraud | Risk & investigation | risk_scores, fraud_cases, disputes |
| Reconciliation | Audit & verification | reconciliation_logs, journal_adjustments, audit_logs |
| Reporting | Analytics | report_definitions, generated_reports, dashboard_metrics |

---

### 8. AUDIT & COMPLIANCE LOGGING

### Core APIs Needed:
```
GET  /audit-logs              → Query audit trail
POST /audit/archive           → Archive old records
GET  /compliance/certifications → Get compliance proof
```

### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `audit_logs` | Complete trail | log_id, user_id, action, resource_type, before_state, after_state, timestamp, approved_by |
| `compliance_archive` | Record preservation | archive_id, record_type, data_hash, timestamp, retention_until |

### Audit Logging Requirements:

**What Gets Logged:**
- All financial transactions
- All player withdrawals
- All admin actions
- All system changes
- All security events
- All KYC updates
- All dispute resolutions

**Immutability:**
- Audit logs cannot be deleted
- Logs cannot be modified
- Access logged
- Export trail recorded

**Retention:**
- Financial records: 7 years (regulatory)
- Audit logs: 5 years minimum
- Compliance records: 10 years
- Dispute records: 3 years post-resolution

---

## PHASE 3 SUMMARY: FINANCIAL INFRASTRUCTURE COMPONENTS

1. ✅ Advanced Treasury & Double-Entry Ledger
2. ✅ Payment Provider Integration (Deposits/Withdrawals)
3. ✅ AML & Compliance System (KYC, screening)
4. ✅ Fraud Detection & Risk Management (Risk scoring, disputes)
5. ✅ Reconciliation & Solvency Engine (Daily reconciliation, health monitoring)
6. ✅ Payment Reconciliation (Provider statement matching)
7. ✅ Reporting & Analytics Infrastructure (Dashboards, reports)
8. ✅ Audit & Compliance Logging (Immutable trails, retention)

---

# IMPLEMENTATION PRIORITY

## Phase 1 Build Order (Critical Path):
1. User System & Authentication (foundation)
2. Wallet & Token System (enables payments)
3. Treasury & Financial System (tracks money)
4. Match & Gameplay System (core feature)
5. Anti-Cheat & Security (prevents cheating)
6. Progression Systems (keeps players engaged)
7. House Challenge Engine (secondary gameplay)
8. Seasonal System (long-term engagement)
9. Replay & Verification System (dispute resolution)
10. Tournament System (competitive path)
11. Leaderboards & Rankings (social motivation)
12. Achievements & Legacy (rewards)

## Phase 3 Build Order (Financial):
1. Payment Provider Integration (payments work first)
2. Advanced Ledger (track all money)
3. Reconciliation Engine (verify daily)
4. AML & Compliance (legal requirement)
5. Fraud Detection (protect treasury)
6. Audit Logging (compliance trail)
7. Reporting & Analytics (visibility)

---

# CRITICAL ARCHITECTURAL DECISIONS

## Server Authority Rules
- **NEVER** trust client for: rewards, XP, rankings, match results, balance calculations
- **ALWAYS** validate server-side
- **Client** is input and display only

## Financial Integrity Rules
- **Double-entry accounting:** Every transaction has debit + credit
- **Segregated reserves:** Funds never mixed
- **Coverage rule:** Player liabilities ≤ reserves always
- **Immutability:** Financial records never deleted
- **Auditability:** Complete trail maintained

## Data Model Principles
- Event sourcing for financial events
- Immutable audit logs
- Never delete records (archive if needed)
- Timestamp everything (UTC)
- Cryptographic verification where possible

