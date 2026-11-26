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
  const detailLine = document.createElement('div');
  detailLine.style.cssText = 'font-size:11px;opacity:.75;margin-top:.2em';
  banner.appendChild(document.createElement('span'));
  banner.appendChild(detailLine);
  const bannerText = banner.firstChild;
  const started = performance.now();
  const trail = [];
  // detail adds a line to what the page says about what it is doing, with the time since it loaded, so
  // that a user whose page does not connect can say where it stopped.
  function detail(text) {
    trail.push(((performance.now() - started) / 1000).toFixed(1) + 's ' + text);
    detailLine.textContent = trail.slice(-4).join(' \u00b7 ') + ' \u00b7 ' + navigator.userAgent.replace(/^Mozilla\/5\.0 /, '').slice(0, 70);
  }
  function say(text) {
    bannerText.textContent = text || '';
    banner.hidden = !text;
  }

  // The banner goes when the page connects, and with it the times, so the steps stay in a dim line at the
  // foot of the page once the room has drawn, which a tap hides. A user timing a first draw on a phone
  // reads it there.
  const timing = document.createElement('div');
  timing.id = 'cogmerViewTiming';
  timing.style.cssText = 'position:fixed;right:8px;bottom:6px;z-index:8;max-width:92vw;font:11px system-ui;opacity:.6;text-align:right';
  timing.hidden = true;
  timing.addEventListener('click', () => { timing.hidden = true; });
  document.addEventListener('DOMContentLoaded', () => document.body.appendChild(timing));
  function showTiming() {
    timing.textContent = trail.join(' · ');
    timing.hidden = false;
  }

  // The button that asks for the turns before the oldest one shown. It sits above the feed and not in
  // it, since the room's drawing puts each turn at its own place among the feed's children (D-308).
  let roomStream = null;
  const earlierButton = document.createElement('button');
  earlierButton.id = 'cogmerViewEarlier';
  earlierButton.type = 'button';
  earlierButton.style.cssText = 'display:none;margin:12px auto;padding:8px 14px;font:13px system-ui;cursor:pointer';
  earlierButton.addEventListener('click', () => { if (roomStream && roomStream.loadEarlier) roomStream.loadEarlier(); });
  document.addEventListener('DOMContentLoaded', () => {
    const feed = document.getElementById('feed');
    if (feed && feed.parentNode) feed.parentNode.insertBefore(earlierButton, feed);
  });
  function showEarlier(total, count, more) {
    earlierButton.style.display = more ? 'block' : 'none';
    if (more) earlierButton.textContent = 'Show earlier turns (' + Math.max(0, total - count) + ' not shown)';
  }

  let ready;
  let http = null;
  let dialAttempt = null;
  let dataSeen = false;
  let asleep = false;

  function loadModule() {
    if (ready) return ready;
    ready = new Promise((resolve, reject) => {
      window.onTailcatReady = () => { detail('module ready'); resolve(); };
      const go = new Go();
      fetch('main.wasm')
        .then((r) => { detail('module fetched'); return WebAssembly.instantiateStreaming(r, go.importObject); })
        .then((r) => go.run(r.instance))
        .catch(reject);
    });
    return ready;
  }

  // startDial dials the daemon, once at a time: the module gives up on its own only after a minute, so
  // the dial in flight is kept until it settles and every caller waits on it. When it succeeds, whenever
  // that is, the connection is kept and the banner goes.
  function startDial() {
    if (!dialAttempt) {
      detail('dialing');
      dialAttempt = tailcatDial({ addr, port: VIEW_PORT, derpMapURL: DERP_MAP })
        .then((conn) => {
          detail('connected');
          http = new ViewConn(conn, token);
          asleep = false;
          say('');
          return http;
        }, (err) => {
          detail('dial failed: ' + err);
          throw err;
        })
        .finally(() => { dialAttempt = null; });
    }
    return dialAttempt;
  }

  // connect gives the connection, dialing if there is none. It says the session is asleep when the dial
  // takes longer than DIAL_WITHIN, and keeps saying so, since every later try finds the same dial, until
  // a dial succeeds.
  function connect() {
    if (http) return Promise.resolve(http);
    if (!addr || !token) return Promise.reject(new Error('this page needs the link from your Claude session'));
    if (!asleep) say('Loading the viewer… the first time this opens on a phone it can take 20 seconds.');
    return loadModule().then(() => {
      if (!asleep) say('Connecting to your session…');
      const limit = new Promise((_, reject) => setTimeout(() => reject(new Error('asleep')), DIAL_WITHIN));
      return Promise.race([startDial(), limit]).catch((err) => {
        if (err.message === 'asleep') {
          asleep = true;
          say('Still connecting to your session. If it is asleep, send a message in its Claude task to wake it; this page keeps trying.');
        } else {
          asleep = true;
          say('Could not reach your session (' + err + '). If it is asleep, send a message in its Claude task to wake it; this page keeps trying.');
        }
        throw err;
      });
    });
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
    // A room's state is asked for with the hash of the one this page holds, and the daemon answers that
    // it is unchanged in a few bytes when it is, with only the age of the lists, which moves as the room
    // sits quiet (D-307). The page puts that age into the state it holds.
    //
    // The daemon sends the newest turns of a room and not all of them, and the turns before the oldest the
    // page holds when it asks (D-308). The page keeps every turn it has fetched, each with the place it
    // sorts in, and hands them to the room's drawing as one list. A newer page replaces the turns from its
    // first on, and an earlier page goes in front.
    const isRoom = target !== null && target.startsWith('/snapshot?');
    let state = null;
    let etag = '';
    let lastLists = '';
    let loaded = [];
    let more = false;
    let firstState = true;
    let noArrival = false;
    function composed() {
      const s = Object.assign({}, state, { events: loaded, noArrival: noArrival });
      noArrival = false;
      return JSON.stringify(s);
    }
    function emit() {
      if (es.onmessage) es.onmessage({ data: composed() });
      showEarlier(state ? state.total : 0, loaded.length, more);
    }
    if (isRoom) {
      es.loadEarlier = async () => {
        if (!loaded.length) return;
        earlierButton.disabled = true;
        try {
          const first = loaded[0].o;
          const r = await request(target + '&before=' + encodeURIComponent(first));
          if (r.status !== 200) throw new Error('status ' + r.status);
          const page = JSON.parse(dec.decode(r.body));
          loaded = (page.events || []).filter((e) => e.o < first).concat(loaded);
          more = !!page.hasEarlier;
          noArrival = true;
          const y = window.scrollY;
          const h = document.documentElement.scrollHeight;
          emit();
          window.scrollTo(0, y + document.documentElement.scrollHeight - h);
        } catch (err) {
          earlierButton.textContent = 'Could not load earlier turns. Tap to try again.';
        } finally {
          earlierButton.disabled = false;
        }
      };
      roomStream = es;
    }
    async function tick() {
      if (es.closed) return;
      try {
        const r = await request(isRoom && etag ? target + '&since=' + etag : target);
        if (r.status !== 200) throw new Error('status ' + r.status);
        if (es.closed) return;
        if (!opened) { opened = true; if (!dataSeen) { dataSeen = true; detail('first data'); showTiming(); } if (es.onopen) es.onopen(); }
        const text = dec.decode(r.body);
        if (isRoom) {
          const got = JSON.parse(text);
          if (got.unchanged && state) {
            const lists = JSON.stringify(got.lists || null);
            if (lists !== lastLists) {
              lastLists = lists;
              if (got.lists) state.lists = got.lists; else delete state.lists;
              emit();
            }
          } else {
            state = got;
            etag = got.etag || '';
            lastLists = JSON.stringify(got.lists || null);
            const win = got.events || [];
            if (win.length) loaded = loaded.filter((e) => e.o < win[0].o).concat(win);
            else loaded = [];
            // The turns the page holds that no newer page carries follow what the room says of them: a
            // turn that appears nowhere goes, and one that does not count is marked.
            if (got.hidden && got.hidden.length) {
              const hidden = new Set(got.hidden);
              loaded = loaded.filter((e) => !hidden.has(e.eventId));
            }
            const marks = got.marks || {};
            loaded.forEach((e) => { if (marks[e.eventId]) e.marked = marks[e.eventId]; else delete e.marked; });
            if (firstState) { firstState = false; more = !!got.hasEarlier; }
            const whole = JSON.stringify(got);
            if (whole !== last) { last = whole; emit(); }
          }
        } else if (text !== last) {
          last = text;
          if (es.onmessage) es.onmessage({ data: text });
        }
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

  // A browser on a phone or a tablet freezes a page that is not shown, and the connections it held are
  // dead when the page is shown again, so a page that has been hidden for a while starts again, and so
  // does one that the browser restores from its cache of pages.
  const RELOAD_AFTER_HIDDEN = 15000;
  let hiddenAt = null;
  document.addEventListener('visibilitychange', () => {
    if (document.hidden) {
      hiddenAt = Date.now();
      return;
    }
    if (hiddenAt !== null && Date.now() - hiddenAt > RELOAD_AFTER_HIDDEN) location.reload();
    hiddenAt = null;
  });
  window.addEventListener('pageshow', (ev) => { if (ev.persisted) location.reload(); });

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
