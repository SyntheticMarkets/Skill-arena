# Arena Hub Product Model

Arena Hub is the authenticated player home for Skill Arena. A player logs into Skill Arena, not into an individual game.

## Arena Hub Owns

- Wallet
- Deposits
- Withdrawals
- Profile
- Avatar
- Overall XP
- Overall level
- Trust score
- Notifications
- Friends
- Houses
- Platform challenges
- Shop
- Settings
- Support

Game modules must not show or own wallet, deposits, withdrawals, KYC, treasury, or account security flows.

## Games Are Modules

Games are applications inside Skill Arena.

Current and future modules:

- Maze Arena
- Memory Arena
- Reaction Arena
- Logic Arena
- Chess Arena
- Sudoku Arena

When a player enters Maze Arena, they remain inside Skill Arena. They enter a focused game module with Maze-specific modes, stats, rankings, achievements, and replays.

## Maze Owns

- Maze home
- Practice
- Ranked
- Tournament play
- Maze replay
- Maze statistics
- Maze achievements
- Maze leaderboard

Maze does not own wallet, deposits, withdrawals, or profile security.

## Progression Split

Arena Hub has overall progression:

- overall level
- overall XP
- trust
- house
- season standing

Each game has game-specific progression:

- Maze level
- Maze rank
- Maze league
- Maze personal bests
- Maze achievements

Future games follow the same model.

## Leaderboards

Arena Hub leaderboards:

- overall players
- houses
- overall XP
- overall season ranking

Game leaderboards:

- Maze global
- Maze weekly
- Maze season
- Maze country

## Challenges

Arena challenges:

- play 3 games
- invite a friend
- complete verification

Game challenges:

- finish Maze under 30 seconds
- reach a combo threshold
- solve a target difficulty

## Navigation Principle

Landing Page -> Authentication -> Arena Hub -> Game Module -> Arena Hub

Back from a game returns to Arena Hub, not to the public landing page.
