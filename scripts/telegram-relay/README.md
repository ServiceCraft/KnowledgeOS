# Telegram Bot API relay (Cloudflare Worker)

Our stage/prod VM (RU-hosted Yandex Cloud) cannot reach `api.telegram.org` —
outbound and inbound Telegram traffic is filtered. Cloudflare's edge reaches
Telegram fine, so we run a tiny pass-through Worker and point the backend at it
via `TELEGRAM_API_BASE_URL`.

```
bot  ──►  https://<worker>.workers.dev/bot<token>/<method>  ──►  https://api.telegram.org/bot<token>/<method>
```

The backend already supports this: set `TELEGRAM_API_BASE_URL` to the worker URL
and (for our blocked VM) `TELEGRAM_POLLING=true`.

## Deploy — Dashboard (no CLI, ~5 min)

1. https://dash.cloudflare.com → **Workers & Pages** → **Create** → **Create Worker**.
2. Name it e.g. `tg-relay`. Create, then **Edit code**.
3. Delete the sample, paste the contents of [`worker.js`](./worker.js), **Deploy**.
4. Copy the worker URL, e.g. `https://tg-relay.<account>.workers.dev`.
5. (Optional hardening) Worker → **Settings → Variables** → add `RELAY_SECRET`
   with a random value, redeploy. Then set the same value on the backend as
   `TELEGRAM_RELAY_SECRET` (see below). Skip for now if you just want it working.

## Deploy — Wrangler (CLI)

```bash
npm i -g wrangler
wrangler login
cd scripts/telegram-relay
wrangler deploy worker.js --name tg-relay
# optional: wrangler secret put RELAY_SECRET
```

## Wire the backend

On the VM in `~/knowledgeos-staging/.env.staging`:

```
TELEGRAM_POLLING=true
TELEGRAM_API_BASE_URL=https://tg-relay.<account>.workers.dev
```

Then recreate the app container. The Telegram adapter will route every Bot API
call (getUpdates, sendMessage, deleteWebhook, …) through the worker.

## Verify

```bash
# from anywhere — should return {"ok":true,...} with the bot info
curl "https://tg-relay.<account>.workers.dev/bot<token>/getMe"
```

If that returns the bot, the VM will reach Telegram through it too.
