// The static page that reads a cloud session's room (D-302). It runs before the view's own script,
// defines the cogmerView object that script looks for, and fills it with what a browser can do:
// dial the daemon through the relay with tailcat's WebAssembly build, speak the port's lines over the
// connection that gives, and poll the daemon's snapshot in place of holding a stream open.
(function () {
  'use strict';
  const enc = new TextEncoder();
  const dec = new TextDecoder();
  function concat(a, b) {
    const out = new Uint8Array(a.length + b.length);
    out.set(a, 0);
    out.set(b, a.length);
    return out;
  }

  // ViewConn speaks the daemon's view port over a connection with read() and write(). The port speaks
  // lines, so the only parsing is a line and a length: the connection opens with the token on a line,
  // each request is a path on a line, and each answer is a line of the status, the media type and the
  // length of the body, then the body. Requests are made one at a time. A request that fails leaves the
  // connection unusable, and the caller dials another.
  class ViewConn {
    constructor(conn, token) {
      this.conn = conn;
      this.buf = new Uint8Array(0);
      this.queue = Promise.resolve();
      this.opened = conn.write(enc.encode(token + '\n'));
    }

    get(path) {
      const run = () => this._get(path);
      const p = this.queue.then(run, run);
      this.queue = p.catch(() => {});
      return p;
    }

    async _fill() {
      const chunk = await this.conn.read();
      if (chunk === null) throw new Error('the connection closed');
      this.buf = concat(this.buf, chunk);
    }

    async _line() {
      for (;;) {
        const i = this.buf.indexOf(10);
        if (i >= 0) {
          const line = dec.decode(this.buf.subarray(0, i));
          this.buf = this.buf.subarray(i + 1);
          return line;
        }
        await this._fill();
      }
    }

    async _take(n) {
      while (this.buf.length < n) await this._fill();
      const out = this.buf.slice(0, n);
      this.buf = this.buf.subarray(n);
      return out;
    }

    async _get(path) {
      await this.opened;
      await this.conn.write(enc.encode(path + '\n'));
      const [status, type, length] = (await this._line()).split(' ');
      const body = await this._take(parseInt(length, 10));
      return { status: parseInt(status, 10), type, body };
    }

    close() {
      try { this.conn.close(); } catch (e) { /* closing a connection that is gone */ }
    }
  }

  // pollTarget is the snapshot a stream's path stands for, or null for a path with none.
  function pollTarget(path) {
    if (path.startsWith('/stream?')) return '/snapshot?' + path.slice('/stream?'.length);
    if (path === '/rooms/stream') return '/rooms';
    return null;
  }

  // ---- the browser -------------------------------------------------------------------------------

  const DERP_MAP = 'https://tailcat.dev/derpmap.json';
  const VIEW_PORT = 4784;
  const DIAL_WITHIN = 15000;
  const REQUEST_WITHIN = 10000;
  const POLL_EVERY = 3000;

  const params = new URLSearchParams(location.hash.slice(1));
  const addr = params.get('a');
  const token = params.get('k');
  const room = params.get('r') || '';

  const banner = document.createElement('div');
  banner.id = 'cogmerViewBanner';
  banner.style.cssText = 'position:fixed;top:0;left:0;right:0;z-index:9;padding:.5em 1em;background:#fff3cd;color:#664d03;font:14px system-ui;text-align:center';
  banner.hidden = true;
  document.addEventListener('DOMContentLoaded', () => {
    document.body.appendChild(banner);
    const exp = document.getElementById('export');
    if (exp) exp.hidden = true;
  });
  function say(text) {
    banner.textContent = text || '';
    banner.hidden = !text;
  }

  let ready;
  let http = null;
  let dialing = null;

  function loadModule() {
    if (ready) return ready;
    ready = new Promise((resolve, reject) => {
      window.onTailcatReady = resolve;
      const go = new Go();
      fetch('main.wasm')
        .then((r) => WebAssembly.instantiateStreaming(r, go.importObject))
        .then((r) => go.run(r.instance))
        .catch(reject);
    });
    return ready;
  }

  // connect dials the daemon, and says the session is asleep when it does not answer in time. One dial
  // is in flight at a time, since the module gives up on its own only after a minute.
  function connect() {
    if (http) return Promise.resolve(http);
    if (dialing) return dialing;
    if (!addr || !token) return Promise.reject(new Error('this page needs the link from your Claude session'));
    say('Loading the viewer…');
    dialing = loadModule().then(() => {
      say('Connecting to your session…');
      const dial = tailcatDial({ addr, port: VIEW_PORT, derpMapURL: DERP_MAP });
      const timer = new Promise((_, reject) => setTimeout(() => reject(new Error('asleep')), DIAL_WITHIN));
      return Promise.race([dial, timer]).then(
        (conn) => { http = new ViewConn(conn, token); say(''); return http; },
        (err) => {
          if (err.message === 'asleep') {
            say('This session looks asleep. Send a message in its Claude task to wake it; this page keeps trying.');
            dial.then((conn) => { http = new ViewConn(conn, token); say(''); }, () => {});
          } else {
            say('Could not reach your session: ' + err);
          }
          throw err;
        });
    }).finally(() => { dialing = null; });
    return dialing;
  }

  // withTimeout gives a request on a connection that is open a limit of its own. A sleeping session
  // answers nothing and closes nothing, so without it a read waits for ever and the page keeps saying it
  // is live.
  function withTimeout(promise) {
    let timer;
    const limit = new Promise((_, reject) => {
      timer = setTimeout(() => reject(new Error('the session did not answer')), REQUEST_WITHIN);
    });
    return Promise.race([promise, limit]).finally(() => clearTimeout(timer));
  }

  function request(path) {
    return connect().then((h) => withTimeout(h.get(path))).catch((err) => {
      if (http) { http.close(); http = null; }
      throw err;
    });
  }

  function stream(path) {
    const target = pollTarget(path);
    const es = { onopen: null, onmessage: null, onerror: null, closed: false, close() { this.closed = true; clearTimeout(timer); } };
    let timer = null;
    let opened = false;
    let last = null;
    async function tick() {
      if (es.closed) return;
      try {
        const r = await request(target);
        if (r.status !== 200) throw new Error('status ' + r.status);
        if (es.closed) return;
        if (!opened) { opened = true; if (es.onopen) es.onopen(); }
        const text = dec.decode(r.body);
        if (text !== last) { last = text; if (es.onmessage) es.onmessage({ data: text }); }
        timer = setTimeout(tick, POLL_EVERY);
      } catch (err) {
        if (es.closed) return;
        if (es.onerror) es.onerror(err);
      }
    }
    if (target === null) setTimeout(() => es.onerror && es.onerror(new Error('no such stream')), 0);
    else setTimeout(tick, 0);
    return es;
  }

  // A pasted image is an <img> whose address is a route of the daemon, which the browser cannot fetch,
  // so each is fetched through the connection and shown from the bytes.
  function showImages(root) {
    root.querySelectorAll('img[src^="/image/"]').forEach((img) => {
      const path = img.getAttribute('src');
      img.removeAttribute('src');
      request(path).then((r) => {
        if (r.status === 200) img.src = URL.createObjectURL(new Blob([r.body], { type: r.type || 'image/png' }));
      }).catch(() => {});
    });
  }
  document.addEventListener('DOMContentLoaded', () => {
    const feed = document.getElementById('feed');
    if (feed) new MutationObserver(() => showImages(feed)).observe(feed, { childList: true, subtree: true });
  });

  window.addEventListener('hashchange', () => location.reload());

  window.cogmerView = {
    room,
    stream,
    roomHref(name) {
      const p = new URLSearchParams({ a: addr, k: token, r: name });
      return location.pathname + '#' + p.toString();
    },
  };

  if (!addr || !token) {
    document.addEventListener('DOMContentLoaded', () => say('Open this page from the link a /cogmer: command gives in your Claude session.'));
  }
})();
