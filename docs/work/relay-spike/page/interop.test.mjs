// Runs the browser client in Node against a relay and responder started by hand, to check that the
// JavaScript and Go handshakes agree: node interop.test.mjs <relay ws url> <responder key>
import { connect, b64url } from './client.js';
const [url, key] = process.argv.slice(2);
const t0 = performance.now();
const conn = await connect(url + '/dial?key=' + key, b64url(key), 'spike', m => console.log(((performance.now() - t0) / 1000).toFixed(3) + 's', m));
const { status, body } = await conn.request('/room');
console.log('status', status, 'bytes', body.length, 'turns', JSON.parse(new TextDecoder().decode(body)).turns.length);
const bad = await conn.request('/nothing');
console.log('status', bad.status);
conn.close();
