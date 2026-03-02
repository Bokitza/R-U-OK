# 🤖 WhatsApp RUOK Bot

> ✅ *"כולם בסדר?"* — An emergency roll-call bot for WhatsApp groups

A Go application that monitors WhatsApp emergency groups. When a group admin asks **"כולם בסדר?"** *(is everyone OK?)*, the bot tracks responses from all participants until everyone has checked in.

---

## 🔄 How It Works

1. 📲 Add the bot to a WhatsApp group via [WasenderAPI](https://wasenderapi.com)
2. 🗣️ A group admin sends **כולם בסדר?**
3. 📢 The bot announces the roll-call event and starts monitoring
4. ⏱️ Every minute, the bot posts a status update showing who responded and who hasn't
5. 🎉 Once everyone responds, the bot sends a summary and closes the event
6. 🔁 Starting a new event in the same group replaces the previous one

---

## 🚀 Setup

### 📋 Prerequisites

| Requirement | Details |
|---|---|
| **Go** | 1.22+ |
| **PostgreSQL** | [Neon](https://neon.tech) or any other PG database |
| **WasenderAPI** | [Account](https://wasenderapi.com) with an active WhatsApp session |

### ⚙️ Configuration

Copy the example env file and fill in your values:

```bash
cp .env.example .env
```

| Variable | Description |
|---|---|
| `WASENDER_API_KEY` | 🔑 Your WasenderAPI session API key |
| `WASENDER_WEBHOOK_SECRET` | 🔐 Secret for verifying webhook requests |
| `DATABASE_URL` | 🗄️ Neon PostgreSQL connection string |
| `BOT_PHONE` | 📱 The bot's phone number (digits only, e.g. `972501234567`) |
| `PORT` | 🌐 HTTP server port (default `8080`) |

### ▶️ Run

```bash
go mod tidy
go run .
```

### 🔗 Webhook Setup

In your WasenderAPI dashboard, set the webhook URL to:

```
https://your-domain.com/webhook
```

Subscribe to the **messages-group.received** event.

### 📡 Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/webhook` | Receives WasenderAPI webhook events |
| `GET` | `/health` | Health check |

---

## ☁️ One-Click Deploy with Fly.io

Deploy this bot to production in seconds using [Fly.io](https://fly.io/). A `fly.toml` is included to make this easy!

**1.** Install the CLI and sign up:

```bash
# https://fly.io/docs/hands-on/install-flyctl/
curl -L https://fly.io/install.sh | sh
```

**2.** Launch:

```bash
fly launch
```

**3.** Set your secrets:

```bash
fly secrets set \
  WASENDER_API_KEY=... \
  WASENDER_WEBHOOK_SECRET=... \
  DATABASE_URL=... \
  BOT_PHONE=...
```

**4.** Deploy:

```bash
fly deploy
```

🎯 Your bot will be running globally at `https://<your-fly-app-name>.fly.dev/webhook`.

For updates, just push changes and run `fly deploy` again. See the [Fly.io Docs](https://fly.io/docs/) for more info.

---

## 🔒 Privacy Notice

> ⚠️ This bot stores phone numbers and message content as part of its core operation — saved in the database for tracking participation and responses during events.

**🧑‍💻 Developer's Responsibility**
It is your responsibility as a developer to safeguard this data. Ensure you do not misuse or improperly disclose phone numbers and messages. Follow all relevant privacy regulations and best practices to protect user information.

**👤 User's Responsibility**
By participating in a group where this bot is active, users implicitly consent to have their phone numbers and response messages stored for event management purposes. Please do not use the bot if you do not agree to these terms.

---

<p align="center">Made with ❤️ for community safety</p>
