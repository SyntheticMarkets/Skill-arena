# Site Map

Status: Draft for review

Sprint: 1 - Product Identity & Design Foundation

## Product Order

Backend complete.

Frontend/product order:

1. Product Identity & Design Foundation
2. Design System
3. Landing
4. Authentication
5. Dashboard
6. Wallet
7. Game Hub
8. Maze Arena
9. Challenges
10. Ranked
11. Leaderboards
12. Tournaments
13. Profile
14. Settings
15. Admin
16. Polish
17. Launch

## Product Design Workflow

Every sprint after this foundation should follow:

```text
Product Documents
  -> Wireframes
  -> High-Fidelity UX Mockups
  -> Approval
  -> Implementation
  -> Testing
  -> Review
  -> Fixes
  -> Commit
  -> Tag
```

Implementation should not begin while the product story, wireframes, or high-fidelity UX mockups are still unresolved.

## Primary Site Structure

```text
Landing
  -> Register
  -> Login
  -> Forgot Password
  -> Terms
  -> Privacy

Authentication
  -> Register
  -> Email Verification
  -> Login
  -> MFA
  -> Password Reset
  -> Age Verification

Authenticated App Shell
  -> Dashboard
  -> Wallet
  -> Game Hub
  -> Challenges
  -> Ranked
  -> Leaderboards
  -> Tournaments
  -> Replays
  -> Profile
  -> Settings
  -> Admin
```

## Authenticated Navigation

```text
Dashboard
  -> Continue Playing
  -> Daily Challenge
  -> Game Hub
  -> Wallet Summary
  -> Leaderboard Preview
  -> Season Progress
  -> Notifications
  -> Recent Games
  -> Achievements

Wallet
  -> Overview
  -> Deposit
  -> Payment Method
  -> Deposit Confirmation
  -> Pending Deposit
  -> Completed Deposit
  -> Withdraw
  -> Pending Withdrawal
  -> Transaction History
  -> Statements
  -> Export

Game Hub
  -> Practice
  -> Ranked
  -> Tournament
  -> Challenge
  -> Training
  -> Future Game Cards
  -> Matchmaking
  -> Live Match
  -> Match Summary
  -> Replay

Maze Arena
  -> Game Rules
  -> Puzzle Board
  -> Game-Specific Controls
  -> Maze Replay Renderer

Matchmaking
  -> Mode Selection
  -> Eligibility Check
  -> Queue
  -> Opponent Found
  -> Countdown
  -> Live Match
  -> Disconnect/Reconnect

Live Match
  -> Game Renderer
  -> Opponent Progress
  -> Timer
  -> Rules State
  -> Victory
  -> Defeat
  -> Match Summary
  -> Replay

Challenges
  -> House Challenges
  -> Daily
  -> Weekly
  -> Monthly
  -> Rewards

Ranked
  -> Queue
  -> Matchmaking
  -> Placement
  -> League
  -> MMR
  -> Promotion
  -> Demotion
  -> History

Leaderboards
  -> Global
  -> Country
  -> Friends
  -> Season
  -> Weekly
  -> Monthly
  -> Search
  -> Filters

Tournaments
  -> List
  -> Detail
  -> Join
  -> Bracket
  -> Matchmaking
  -> Live Match
  -> Results
  -> Replay

Profile
  -> Stats
  -> History
  -> Achievements
  -> Trust
  -> Badges
  -> Avatar
  -> Customization

Settings
  -> Profile
  -> Security
  -> Wallet
  -> Notifications
  -> Language
  -> Accessibility
  -> Privacy

Admin
  -> Dashboard
  -> Users
  -> Wallet
  -> Treasury
  -> Games
  -> Moderation
  -> Support
  -> Reports
  -> Analytics
```

## Player Movement Model

Primary loop:

```text
Dashboard -> Game Hub -> Matchmaking/Challenge -> Live Match -> Summary -> Replay/Rematch -> Dashboard
```

Money loop:

```text
Dashboard -> Wallet -> Deposit -> Game Hub -> Live Match -> Wallet -> Withdraw
```

Competitive loop:

```text
Dashboard -> Ranked -> Matchmaking -> Live Match -> Rank Movement -> Leaderboard -> Queue Again
```

Tournament loop:

```text
Dashboard -> Tournaments -> Join -> Bracket -> Matchmaking -> Live Match -> Results -> Replay -> Next Round
```

Trust loop:

```text
Dashboard -> Verification/Security -> Calibration -> Clean Matches -> Trust Score Increase -> Higher Access
```

## Navigation Priority

Top-level navigation should prioritize:

1. Dashboard
2. Play/Game Hub
3. Wallet
4. Ranked
5. Tournaments
6. Leaderboards
7. Profile
8. Settings

Admin navigation should only appear for privileged roles.

## Approval Questions

1. Does this site map put the platform before Maze Arena?
2. Does every major path lead toward play, competition, trust, or progress?
3. Are wallet and ranked flows accessible without overwhelming new users?
4. Can future games fit without restructuring global navigation?
5. Are matchmaking and live match flows game-agnostic?
