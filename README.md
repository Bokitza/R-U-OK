# WhatsApp RUOK Bot

A Go application that monitors WhatsApp emergency groups. When a group admin asks "כולם בסדר?" (is everyone OK?), the bot tracks responses from all participants until everyone has checked in.

## How It Works

1. Add the bot to a WhatsApp group via [WasenderAPI](https://wasenderapi.com)
2. A group admin sends **כולם בסדר?**
3. The bot announces the roll-call event and starts monitoring
4. Every minute, the bot posts a status update showing who responded and who hasn't
5. Once everyone responds, the bot sends a summary and closes the event
6. Starting a new event in the same group replaces the previous one

## Setup

### Prerequisites

- Go 1.22+
- A [Neon](https://neon.tech) PostgreSQL database (or plainly any other PG DB)
- A [WasenderAPI](https://wasenderapi.com) account with an active WhatsApp session

### Configuration

Copy the example env file and fill in your values:

```bash
cp .env.example .env
```

| Variable | Description |
|----------|-------------|
| `WASENDER_API_KEY` | Your WasenderAPI session API key |
| `WASENDER_WEBHOOK_SECRET` | Secret for verifying webhook requests |
| `DATABASE_URL` | Neon PostgreSQL connection string |
| `BOT_PHONE` | The bot's phone number (digits only, e.g. `972501234567`) |
| `PORT` | HTTP server port (default `8080`) |

### Run

```bash
go mod tidy
go run .
```

### Webhook Setup

In your WasenderAPI dashboard, set the webhook URL to:

```
https://your-domain.com/webhook
```

Subscribe to the **messages-group.received** event.

### Deploy

The bot exposes two endpoints:

- `POST /webhook` — receives WasenderAPI webhook events
- `GET /health` — health check

## Privacy Notice

This bot stores phone numbers and message content as part of its core operation—these are saved in the database for tracking participation and responses during events.

**Developer's Responsibility:**  
It is your responsibility as a developer to safeguard this data. Ensure you do not misuse or improperly disclose phone numbers and messages. Follow all relevant privacy regulations and best practices to protect user information.

**User's Responsibility:**  
By participating in a group where this bot is active, users implicitly consent to have their phone numbers and response messages stored for event management purposes. Please do not use the bot if you do not agree to these terms.
