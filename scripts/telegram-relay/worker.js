/**
 * Telegram Bot API relay — Cloudflare Worker.
 *
 * Why: our stage/prod VM is on RU-hosted Yandex Cloud, where outbound (and
 * inbound) traffic to api.telegram.org is fully filtered. Cloudflare's edge can
 * reach Telegram fine, so the bot points TELEGRAM_API_BASE_URL at this worker and
 * the worker forwards every request through to Telegram unchanged.
 *
 * The bot calls:  https://<worker-host>/bot<token>/<method>
 * The worker sends: https://api.telegram.org/bot<token>/<method>
 *
 * Optional hardening: set a RELAY_SECRET var (Worker → Settings → Variables).
 * When set, the bot must send it as the `X-Relay-Secret` header (configure the
 * same value on the backend) or the worker rejects the request. Leave it unset
 * to run open — the bot token in the path is already required by Telegram.
 */

const TELEGRAM_ORIGIN = 'https://api.telegram.org';

export default {
  async fetch(request, env) {
    const url = new URL(request.url);

    // Only proxy Bot API calls (path starts with /bot<token>/).
    if (!url.pathname.startsWith('/bot')) {
      return new Response('Not found', { status: 404 });
    }

    // Optional shared-secret gate.
    if (env.RELAY_SECRET) {
      if (request.headers.get('X-Relay-Secret') !== env.RELAY_SECRET) {
        return new Response('Forbidden', { status: 403 });
      }
    }

    const target = TELEGRAM_ORIGIN + url.pathname + url.search;

    // Rebuild the request for the upstream, dropping hop-by-hop / relay headers.
    const headers = new Headers(request.headers);
    headers.delete('host');
    headers.delete('x-relay-secret');

    const init = {
      method: request.method,
      headers,
      body:
        request.method === 'GET' || request.method === 'HEAD'
          ? undefined
          : request.body,
    };

    // Long-polling getUpdates can block server-side; Cloudflare allows long
    // subrequests, so just forward and stream the response back.
    const upstream = await fetch(target, init);
    const respHeaders = new Headers(upstream.headers);
    return new Response(upstream.body, {
      status: upstream.status,
      statusText: upstream.statusText,
      headers: respHeaders,
    });
  },
};
