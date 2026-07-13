# Session Gateway

Session Gateway is the single authenticated live connection for a logged-in client.

The client opens one WebSocket after login:

1. Login with REST.
2. Receive JWT.
3. Open one WebSocket.
4. Authenticate the socket with the JWT.
5. Receive all live events through that connection.

## REST Is For

- login
- register
- deposit
- withdraw
- settings
- profile
- avatar
- KYC
- password reset
- account security

These are request and response flows.

## WebSocket Is For

- matchmaking
- match found
- PvP countdown
- timers
- opponent progress
- replay spectating
- leaderboard movement
- notifications
- challenge progress
- tournament updates
- presence

These are live flows.

## Event Path

Game module -> Arena Core -> Event Bus -> Session Gateway -> Client

The client still sends intent only. The server remains authoritative.

Example:

Client sends:

```json
{
  "type": "game.action",
  "sessionId": "sess_123",
  "action": {
    "actionType": "click",
    "targetId": "line_17",
    "sequence": 4
  }
}
```

Server responds with authoritative state/event:

```json
{
  "type": "action_accepted",
  "scope": "game",
  "scopeId": "sess_123",
  "payload": {
    "accepted": true,
    "progress": 42
  }
}
```

The client must never send wallet changes, rewards, winner, score, trust, XP, or completion state.

## One Connection

Do not create one WebSocket per feature or game. One authenticated Session Gateway connection carries:

- game events
- matchmaking events
- tournament events
- notifications
- challenge progress
- presence
- live leaderboard updates

This keeps reconnect, presence, and authorization simple.
