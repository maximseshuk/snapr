# Telegram

`type: telegram` — sends a message via the [Telegram Bot API](https://core.telegram.org/bots/api).

## Fields

| Field      | Type       | Required | Notes                                                        |
| ---------- | ---------- | -------- | ------------------------------------------------------------ |
| `type`     | `telegram` | yes      |                                                              |
| `botToken` | string     | yes      | token from [@BotFather](https://t.me/BotFather) (use `env:`) |
| `chatId`   | string     | yes      | numeric chat ID (negative for groups)                        |

Plus the [common fields](/configuration/notifiers/).

## Example

```yaml
notifiers:
  - type: telegram
    name: oncall
    botToken: env:TG_BOT_TOKEN
    chatId: '-1001234567890'
    onFailure: true
```

## Tips

- To find a chat ID, message the bot then call `getUpdates` once.
- The bot must be a member of the target group/channel before it can post.
