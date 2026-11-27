// The browser's side of the relay spike: Noise_NK_25519_AESGCM_SHA256 with WebCrypto, over one WebSocket
// to the relay, then D-301's lines. It holds no library. The Go side is relay-spike/noise.
const enc = new TextEncoder(), dec = new TextDecoder();
const subtle = globalThis.crypto.subtle;
const NAME = 'Noise_NK_25519_AESGCM_SHA256';

const cat = (...a) => { const o = new Uint8Array(a.reduce((n, x) => n + x.length, 0)); let i = 0; for (const x of a) { o.set(x, i); i += x.length; } return o; };
const sha256 = async d => new Uint8Array(await subtle.digest('SHA-256', d));
const hmac = async (k, d) => new Uint8Array(await subtle.sign('HMAC', await subtle.importKey('raw', k, { name: 'HMAC', hash: 'SHA-256' }, false, ['sign']), d));
async function hkdf2(ck, ikm) { const t = await hmac(ck, ikm); const a = await hmac(t, Uint8Array.of(1)); const b = await hmac(t, cat(a, Uint8Array.of(2))); return [a, b]; }
const nonce = n => { const b = new Uint8Array(12); new DataView(b.buffer).setBigUint64(4, BigInt(n)); return b; };
const aes = k => subtle.importKey('raw', k, 'AES-GCM', false, ['encrypt', 'decrypt']);
async function seal(k, n, ad, p) { return new Uint8Array(await subtle.encrypt({ name: 'AES-GCM', iv: nonce(n), additionalData: ad }, await aes(k), p)); }
async function open(k, n, ad, c) { return new Uint8Array(await subtle.decrypt({ name: 'AES-GCM', iv: nonce(n), additionalData: ad }, await aes(k), c)); }

export function b64url(s) { s = s.replace(/-/g, '+').replace(/_/g, '/'); while (s.length % 4) s += '='; return Uint8Array.from(atob(s), c => c.charCodeAt(0)); }

export async function supported() {
  try { await subtle.generateKey({ name: 'X25519' }, true, ['deriveBits']); return true; } catch { return false; }
}

// connect dials the daemon whose static public key is rs through the relay at url, and returns a
// function that sends a request path and resolves to its status line and body.
export async function connect(url, rs, token, mark = () => {}) {
  const ws = new WebSocket(url);
  ws.binaryType = 'arraybuffer';
  const inbox = [], waiters = [];
  ws.onmessage = ev => { const m = new Uint8Array(ev.data); waiters.length ? waiters.shift()(m) : inbox.push(m); };
  let failed = null;
  ws.onclose = ev => { failed = new Error('the relay closed the connection: ' + (ev.reason || ev.code)); while (waiters.length) waiters.shift()(null); };
  const next = () => inbox.length ? Promise.resolve(inbox.shift()) : new Promise(r => waiters.push(r));
  await new Promise((res, rej) => { ws.onopen = res; ws.onerror = () => rej(new Error('could not reach the relay')); });
  mark('connected');

  let h = new Uint8Array(32); h.set(enc.encode(NAME));
  let ck = h;
  const mixHash = async d => { h = await sha256(cat(h, d)); };
  let k = null, n = 0;
  const mixKey = async ikm => { [ck, k] = await hkdf2(ck, ikm); n = 0; };
  await mixHash(new Uint8Array(0));
  await mixHash(rs);
  const e = await subtle.generateKey({ name: 'X25519' }, true, ['deriveBits']);
  const epub = new Uint8Array(await subtle.exportKey('raw', e.publicKey));
  await mixHash(epub);
  const rsKey = await subtle.importKey('raw', rs, { name: 'X25519' }, false, []);
  await mixKey(new Uint8Array(await subtle.deriveBits({ name: 'X25519', public: rsKey }, e.privateKey, 256)));
  const c1 = await seal(k, n++, h, new Uint8Array(0)); await mixHash(c1);
  ws.send(cat(epub, c1));

  const m2 = await next();
  if (!m2) throw failed;
  const re = m2.slice(0, 32);
  await mixHash(re);
  const reKey = await subtle.importKey('raw', re, { name: 'X25519' }, false, []);
  await mixKey(new Uint8Array(await subtle.deriveBits({ name: 'X25519', public: reKey }, e.privateKey, 256)));
  await open(k, n++, h, m2.slice(32)); // throws when the daemon does not hold the key the link names
  await mixHash(m2.slice(32));
  const [sendK, recvK] = await hkdf2(ck, new Uint8Array(0));
  let sn = 0, rn = 0;
  mark('secure');

  const send = async s => ws.send(await seal(sendK, sn++, new Uint8Array(0), enc.encode(s + '\n')));
  const recv = async () => { const m = await next(); if (!m) throw failed; return open(recvK, rn++, new Uint8Array(0), m); };
  await send(token);
  return {
    async request(path) {
      await send(path);
      const status = dec.decode(await recv()).trim();
      const body = await recv();
      return { status, body };
    },
    close() { ws.close(); },
  };
}
