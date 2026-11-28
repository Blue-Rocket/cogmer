"""Monte Carlo simulation of how cogmer's capture and injection options fare when the
vendors change the capabilities they rest on.

Run with:  python3 -I sim.py params.json OUTDIR [--checks-only] [--runs N]

What it does, in order:
  1. Draws, for each run, every break of every vendor capability over 24 months: a
     per-capability monthly hazard (silent-absent, silent-corrupt, loud), a removal
     hazard, and vendor-wide redesign shocks that break many capabilities at once.
     Each break carries pre-drawn delays (doctor, users, noise, guard, adaptation,
     vendor fix, replacement), so every portfolio sees the same world (common random
     numbers), and differences between portfolios are not noise from separate draws.
  2. For each portfolio (one capture rule and one injection rule on one host),
     computes when each break is detected through the channels that portfolio has,
     when it ends, and samples capture and injection at 4 points a month.
  3. Reports, per portfolio, the fraction of time capture and injection work, the
     months of silent failure, and engineering cost, with 95% intervals.
Standard library only. Writes results.json and results.csv in OUTDIR.
"""
import json, math, random, sys, os, statistics, copy, time

OK, ABSENT, CORRUPT = 'ok', 'absent', 'corrupt'
INF = float('inf')

# ---------------------------------------------------------------- capture rules
# Each takes s: cap -> state and returns 'full', 'partial', 'none' or 'wrong'.

def _ids_prompt(s, ids, prompt):
    for c in (ids, prompt):
        if s[c] == CORRUPT:
            return 'wrong'
        if s[c] == ABSENT:
            return 'none'
    return None

def two_source(s, a, b):
    """Union of two sources of the reply (D-003): both -> full, one -> partial."""
    if s[a] == CORRUPT or s[b] == CORRUPT:
        return 'wrong'
    n = (s[a] == OK) + (s[b] == OK)
    return 'full' if n == 2 else 'partial' if n == 1 else 'none'

def cap_cc_union(s):
    return _ids_prompt(s, 'A_ids', 'A_prompt') or two_source(s, 'A_stop', 'A_transcript')

def cap_cc_payload(s):
    r = _ids_prompt(s, 'A_ids', 'A_prompt')
    if r: return r
    return {OK: 'partial', ABSENT: 'none', CORRUPT: 'wrong'}[s['A_stop']]

def cap_cc_msgdisp(s):
    return _ids_prompt(s, 'A_ids', 'A_prompt') or two_source(s, 'A_stop', 'A_msgdisp')

def cap_cc_three(s):
    """Stop payload + transcript + MessageDisplay: full when the final text and
    either pre-tool source are present."""
    r = _ids_prompt(s, 'A_ids', 'A_prompt')
    if r: return r
    if CORRUPT in (s['A_stop'], s['A_transcript'], s['A_msgdisp']):
        return 'wrong'
    pre = s['A_transcript'] == OK or s['A_msgdisp'] == OK
    if s['A_stop'] == OK:
        return 'full' if pre else 'partial'
    return 'partial' if pre else 'none'

def cap_cc_otel(s):
    if s['A_ids'] != OK:   # binding a session to a room still comes from the hooks
        return 'none' if s['A_ids'] == ABSENT else 'wrong'
    return {OK: 'full', ABSENT: 'none', CORRUPT: 'wrong'}[s['A_otel']]

def gate(cap, rule):
    def f(s):
        if s[cap] == ABSENT: return 'none'
        if s[cap] == CORRUPT: return 'wrong'
        return rule(s)
    f.__name__ = rule.__name__ + '+' + cap
    return f

def cap_codex_union(s):
    if s['O_ids'] != OK:
        return 'none' if s['O_ids'] == ABSENT else 'wrong'
    return two_source(s, 'O_stop', 'O_rollout')

def cap_codex_payload(s):
    if s['O_ids'] != OK:
        return 'none' if s['O_ids'] == ABSENT else 'wrong'
    return {OK: 'partial', ABSENT: 'none', CORRUPT: 'wrong'}[s['O_stop']]

# ---------------------------------------------------------------- injection rules
# Return 'ok', 'noisy' (delivered, but offered again: the re-offer of D-131),
# 'none' (not delivered, and offered again, so it announces itself), or 'lost'
# (counted delivered and never received, or received misframed).

def inj_evidence(ids, inject, cap, evidence):
    def f(s):
        if s[ids] != OK: return 'none' if s[ids] == ABSENT else 'lost'
        if s[inject] == CORRUPT: return 'lost'
        if s[inject] == ABSENT: return 'none'
        if cap and s[cap] != OK: return 'none'   # a preview is recorded, never confirmed (B34)
        return 'ok' if s[evidence] == OK else 'noisy'
    return f

def inj_trust(ids, stoppid, inject, cap):
    def f(s):
        if s[ids] != OK: return 'none' if s[ids] == ABSENT else 'lost'
        if stoppid and s[stoppid] == CORRUPT: return 'lost'
        if s[inject] != OK: return 'lost'
        if cap and s[cap] != OK: return 'lost'   # a preview counted delivered at stop
        if stoppid and s[stoppid] == ABSENT: return 'noisy'   # B29: offered again
        return 'ok'
    return f

def inj_evidence_else_trust(ids, inject, cap, evidence):
    """Evidence when the record is there; when it is not, count at stop (D-131's trust
    once the check fails, applied as soon as the evidence is missing)."""
    ev = inj_evidence(ids, inject, cap, evidence)
    tr = inj_trust(ids, None, inject, cap)
    def f(s):
        return ev(s) if s[evidence] == OK else tr(s)
    return f

def ginj(cap, rule):
    def f(s):
        if s[cap] == ABSENT: return 'none'
        if s[cap] == CORRUPT: return 'lost'
        return rule(s)
    return f

def ginj_lost(cap, rule):
    """A gate whose absence is silent loss: the Cowork bridge's reply not reaching the
    model while the stop call still arrives (B30, D-206 Limits)."""
    def f(s):
        if s[cap] != OK: return 'lost'
        return rule(s)
    return f

CC_EV = inj_evidence('A_ids', 'A_inject', 'A_cap', 'A_evidence')
CC_TRUST = inj_trust('A_ids', 'A_stoppid', 'A_inject', 'A_cap')
CX_EV = inj_evidence('O_ids', 'O_inject', None, 'O_rollout')
CX_TRUST = inj_trust('O_ids', None, 'O_inject', None)
CX_EV_ELSE = inj_evidence_else_trust('O_ids', 'O_inject', None, 'O_rollout')

# ---------------------------------------------------------------- portfolios
# caps: every capability the portfolio's rules read. noisy: capabilities whose
# failure the portfolio announces through re-offering (users see colleagues' turns
# again or late). guard: transcript or rollout capabilities whose corruption a
# cross-check with last_assistant_message turns into a known absence.
# build_pw: person-weeks to build from today's code (assumed).

def P(id, host, name, capture, inject, caps, noisy=(), guard=(), build=0.0, status='', values=''):
    return dict(id=id, host=host, name=name, capture=capture, inject=inject, caps=list(caps),
                noisy=set(noisy), guard=set(guard), build=build, status=status, values=values)

CC_CAPS = ['A_ids', 'A_prompt', 'A_stop', 'A_transcript', 'A_inject', 'A_cap', 'A_evidence']
PORTFOLIOS = [
    P('CC1', 'cc_local', 'current: hooks + transcript union, transcript evidence', cap_cc_union, CC_EV,
      CC_CAPS, noisy=['A_inject', 'A_cap', 'A_evidence'], status='current (D-003, D-014, D-131)', values='meets'),
    P('CC2', 'cc_local', 'hook payloads only, delivery counted at Stop', cap_cc_payload, CC_TRUST,
      ['A_ids', 'A_prompt', 'A_stop', 'A_stoppid', 'A_inject', 'A_cap'], noisy=['A_stoppid'], build=0.5,
      status='rejected for a local session: D-003 (payload alone loses pre-tool text), D-131 (trust without evidence)', values='silent-loss risk'),
    P('CC3', 'cc_local', 'current + transcript cross-checked against last_assistant_message', cap_cc_union, CC_EV,
      CC_CAPS, noisy=['A_inject', 'A_cap', 'A_evidence'], guard=['A_transcript'], build=1.0, status='new', values='meets'),
    P('CC4', 'cc_local', 'Stop + MessageDisplay capture, transcript evidence', cap_cc_msgdisp, CC_EV,
      ['A_ids', 'A_prompt', 'A_stop', 'A_msgdisp', 'A_inject', 'A_cap', 'A_evidence'],
      noisy=['A_inject', 'A_cap', 'A_evidence'], build=2.0, status='new', values='only if the hook adds no latency to display'),
    P('CC5', 'cc_local', 'Stop + MessageDisplay capture, delivery at Stop (no transcript at all)', cap_cc_msgdisp, CC_TRUST,
      ['A_ids', 'A_prompt', 'A_stop', 'A_stoppid', 'A_msgdisp', 'A_inject', 'A_cap'], noisy=['A_stoppid'], build=2.5,
      status='new; injection half rejected by D-131 for a local session', values='silent-loss risk; display latency'),
    P('CC6', 'cc_local', 'guarded union + MessageDisplay as third source, transcript evidence', cap_cc_three, CC_EV,
      CC_CAPS + ['A_msgdisp'], noisy=['A_inject', 'A_cap', 'A_evidence'], guard=['A_transcript'], build=3.0,
      status='new', values='only if the hook adds no latency to display'),
    P('CC7', 'cc_local', 'OpenTelemetry log events for capture, transcript evidence', cap_cc_otel, CC_EV,
      ['A_ids', 'A_otel', 'A_inject', 'A_cap', 'A_evidence'], noisy=['A_inject', 'A_cap', 'A_evidence'], build=3.0,
      status='new', values='fails: needs settings outside the plugin and takes over the exporter an organization may use (D-037)'),
    P('CL1', 'cc_cloud', 'current in a Claude Code cloud session (daemon in container)', gate('A_cloudenv', cap_cc_union),
      ginj('A_cloudenv', CC_EV), CC_CAPS + ['A_cloudenv'], noisy=['A_inject', 'A_cap', 'A_evidence'],
      status='current (D-288, D-297: no checks in a container)', values='meets'),
    P('CL2', 'cc_cloud', 'CL1 + guard', gate('A_cloudenv', cap_cc_union), ginj('A_cloudenv', CC_EV),
      CC_CAPS + ['A_cloudenv'], noisy=['A_inject', 'A_cap', 'A_evidence'], guard=['A_transcript'], build=1.0,
      status='new', values='meets'),
    P('CW1', 'cowork_local', 'bridge: mcp_tool hooks, payload capture, delivery at Stop', gate('A_mcphook', cap_cc_payload),
      ginj_lost('A_mcphook', CC_TRUST), ['A_mcphook', 'A_ids', 'A_prompt', 'A_stop', 'A_stoppid', 'A_inject', 'A_cap'],
      noisy=['A_stoppid'], status='current (D-205, D-206)', values='silent loss accepted in D-206 Limits'),
    P('CW2', 'cowork_local', 'plugin command hooks directly, payload capture, delivery at Stop', cap_cc_payload, CC_TRUST,
      ['A_ids', 'A_prompt', 'A_stop', 'A_stoppid', 'A_inject', 'A_cap'], noisy=['A_stoppid'], build=1.0,
      status='observed working 2026-11-18 (open.md, e7ddab7); commands still need the bridge', values='as CW1'),
    P('CW3', 'cowork_local', 'CW2 + MessageDisplay capture', cap_cc_msgdisp, CC_TRUST,
      ['A_ids', 'A_prompt', 'A_stop', 'A_stoppid', 'A_msgdisp', 'A_inject', 'A_cap'], noisy=['A_stoppid'], build=2.5,
      status='new', values='as CW1; display latency'),
    P('KC1', 'cowork_cloud', 'current: daemon in sandbox, payload capture, delivery at Stop', gate('A_cloudenv', cap_cc_payload),
      ginj('A_cloudenv', CC_TRUST), ['A_cloudenv', 'A_ids', 'A_prompt', 'A_stop', 'A_stoppid', 'A_inject', 'A_cap'],
      noisy=['A_stoppid'], status='current (D-288, D-295, D-296)', values='silent loss accepted in D-296 Limits'),
    P('KC2', 'cowork_cloud', 'KC1 + MessageDisplay capture', gate('A_cloudenv', cap_cc_msgdisp), ginj('A_cloudenv', CC_TRUST),
      ['A_cloudenv', 'A_ids', 'A_prompt', 'A_stop', 'A_stoppid', 'A_msgdisp', 'A_inject', 'A_cap'], noisy=['A_stoppid'],
      build=1.5, status='new', values='as KC1'),
    P('CX1', 'codex_local', 'current: hooks + rollout union, rollout evidence, no trust', cap_codex_union, CX_EV,
      ['O_ids', 'O_stop', 'O_rollout', 'O_inject'], noisy=['O_inject', 'O_rollout'], status='current (D-201, D-202)', values='meets'),
    P('CX2', 'codex_local', 'hook payloads only, delivery at Stop', cap_codex_payload, CX_TRUST,
      ['O_ids', 'O_stop', 'O_inject'], build=0.5, status='rejected for Codex by D-202 Limits (no trust fallback)', values='silent-loss risk'),
    P('CX3', 'codex_local', 'guarded union; rollout evidence, delivery at Stop while the rollout is unreadable', cap_codex_union, CX_EV_ELSE,
      ['O_ids', 'O_stop', 'O_rollout', 'O_inject'], noisy=['O_inject'], guard=['O_rollout'], build=1.5,
      status='new; revises D-202 Limits', values='silent-loss risk only while the rollout is broken'),
    P('XC1', 'codex_cloud', 'command hooks if Codex cloud ever runs them; payload + delivery at Stop', gate('O_cloudhooks', cap_codex_payload),
      ginj('O_cloudhooks', CX_TRUST), ['O_cloudhooks', 'O_ids', 'O_stop', 'O_inject'], build=3.0,
      status='not available today (open.md: Codex cloud task unknown)', values='meets if it arrives'),
]

# ---------------------------------------------------------------- world

def expo(rng, mean):
    return rng.expovariate(1.0 / mean) if mean > 0 else 0.0

class Break:
    __slots__ = ('cap', 'vendor', 'start', 'kind', 'surfaces', 'shock', 'unavail',
                 'd_doc', 'doc_hit', 'd_user', 'd_noisy', 'guard_u', 'd_guard',
                 'd_adapt', 'adapt_ok', 'vfix', 'replace')

def draw_world(prm, rng):
    caps, months = prm['caps'], prm['months']
    D, A, V, IMP = prm['detect'], prm['adapt'], prm['vendor_fix'], prm['improve']
    scope_all = prm.get('scope_all', 0.75)
    breaks = []

    def new_break(cap, start, kind, shock=False, unavail=False):
        c = caps[cap]
        b = Break()
        b.cap, b.vendor, b.start, b.kind, b.shock, b.unavail = cap, c['vendor'], start, kind, shock, unavail
        on = c['on']
        # A removal is a vendor-wide decision and hits every surface the capability is on;
        # a loss on one surface is drawn as an 'absent' break with a narrow scope. (The
        # first version drew removals with a narrow scope too, and the removal sanity
        # check failed on it.)
        b.surfaces = set(on) if (shock or unavail or kind == 'removal' or rng.random() < scope_all) else {rng.choice(on)}
        b.d_doc = expo(rng, D['doctor_mean']); b.doc_hit = rng.random() < D['doctor_coverage']
        loud = kind in ('loud',) or shock
        b.d_user = expo(rng, prm['shock']['detect_mean'] if shock else (D['user_loud_mean'] if loud else D['user_silent_mean']))
        b.d_noisy = expo(rng, D['noisy_mean'])
        b.guard_u = rng.random(); b.d_guard = expo(rng, D['guard_mean'])
        b.d_adapt = expo(rng, prm['shock']['adapt_mean'] if shock else A['change_mean'])
        b.adapt_ok = rng.random() < A['success']
        b.vfix = expo(rng, V['mean']) if (not shock and not unavail and kind != 'removal'
                                          and rng.random() < V['regression_share']) else INF
        b.replace = expo(rng, 1.0 / (IMP['unavailable'] if unavail else IMP['replacement'])) \
            if (IMP['unavailable'] if unavail else IMP['replacement']) > 0 else INF
        breaks.append(b)

    for cap, c in caps.items():
        if c.get('unavailable_at_start'):
            new_break(cap, 0.0, 'removal', unavail=True)
        sp = c['split']; kinds = list(sp); w = [sp[k] for k in kinds]
        for m in range(months):
            if rng.random() < c['hazard']:
                new_break(cap, float(m), rng.choices(kinds, w)[0])
            if rng.random() < c['removal']:
                new_break(cap, float(m), 'removal')
    S = prm['shock']
    sk = list(S['split']); sw = [S['split'][k] for k in sk]
    for vendor in ('A', 'O'):
        for m in range(months):
            if rng.random() < S[vendor]:
                for cap, c in caps.items():
                    if c['vendor'] == vendor and rng.random() < S['hit_prob']:
                        new_break(cap, float(m), rng.choices(sk, sw)[0], shock=True)
    return breaks

# ---------------------------------------------------------------- one portfolio in one world

def timeline(prm, pf, b):
    """Detection time, end time, and the state the break gives, for this portfolio."""
    caps = prm['caps']; D = prm['detect']
    kind = b.kind
    state = ABSENT if kind in ('absent', 'removal', 'loud') else CORRUPT
    loud = kind == 'loud' or b.shock or b.unavail
    lags = [b.d_user]
    if caps[b.cap]['checked'] and b.doc_hit and prm['doctor_surface'][b.vendor] in b.surfaces:
        lags.append(b.d_doc)
    if b.cap in pf['noisy'] and state == ABSENT:
        lags.append(b.d_noisy)
    if b.cap in pf['guard'] and state == CORRUPT and b.guard_u < D['guard_catch']:
        state = ABSENT
        lags.append(b.d_guard)
    det = 0.0 if b.unavail else b.start + min(lags)
    lag = prm['adapt']['release_lag'].get(pf['host'], 0.0)
    if kind == 'removal' or not b.adapt_ok:
        end = max(b.start + b.replace, det) + b.d_adapt + lag if b.replace < INF else INF
    else:
        end = det + b.d_adapt + lag
    end = min(end, b.start + b.vfix)
    return det, end, state, loud

def eval_portfolio(prm, pf, breaks):
    months, ss = prm['months'], prm['substeps']
    caps = set(pf['caps']); host = pf['host']
    rel = [b for b in breaks if b.cap in caps and host in b.surfaces]
    n = months * ss
    # what the rules give with nothing broken: a payload-only rule never gives 'full'
    # (the first version assumed it did, and its 'full' numbers were wrong)
    allok = {c: OK for c in caps}
    full0 = 1.0 if pf['capture'](allok) == 'full' else 0.0
    res = dict(cap_works=1.0, cap_full=full0, inj_works=1.0, inj_clean=1.0,
               silent_cap=0.0, silent_inj=0.0, silent_any=0.0, cost=pf['build'])
    # cost: every break of a used capability that cogmer detects within the horizon,
    # wherever it was seen, costs an adaptation
    C = prm['cost']
    for b in breaks:
        if b.cap in caps and not b.unavail:
            det, end, st, loud = timeline(prm, pf, b)
            if det < months:
                res['cost'] += C['shock_pw'] if b.shock else C['removal_pw'] if b.kind == 'removal' else C['change_pw']
    if not rel:
        return res
    tl = [(b.cap,) + timeline(prm, pf, b) for b in rel]
    base = {c: OK for c in caps}
    cw = cf = iw = ic = 0
    s_cap = s_inj = s_any = 0
    for k in range(n):
        t = (k + 0.5) / ss
        s = dict(base); known = {}
        any_bad = False
        for (cap, det, end, st, loud), b in zip(tl, rel):
            if b.start <= t < end:
                any_bad = True
                if s[cap] != ABSENT:
                    s[cap] = st if s[cap] == OK or st == ABSENT else s[cap]
                if loud or t >= det:
                    known[cap] = True
        if not any_bad:
            cw += 1; cf += full0; iw += 1; ic += 1
            continue
        lc = pf['capture'](s); li = pf['inject'](s)
        cap_fail = lc in ('none', 'wrong'); inj_fail = li in ('none', 'lost')
        cw += not cap_fail; cf += lc == 'full'
        iw += not inj_fail; ic += li == 'ok'
        bad = [c for c in caps if s[c] != OK]
        silent = not any(known.get(c) for c in bad)
        if cap_fail and silent: s_cap += 1
        if inj_fail and silent: s_inj += 1
        if (cap_fail or inj_fail) and silent: s_any += 1
    res.update(cap_works=cw / n, cap_full=cf / n, inj_works=iw / n, inj_clean=ic / n,
               silent_cap=s_cap / ss, silent_inj=s_inj / ss, silent_any=s_any / ss)
    return res

# ---------------------------------------------------------------- driver

METRICS = ['cap_works', 'cap_full', 'inj_works', 'inj_clean', 'silent_cap', 'silent_inj', 'silent_any', 'cost']

def simulate(prm, runs, seed, portfolios=PORTFOLIOS):
    rng = random.Random(seed)
    acc = {p['id']: {m: [] for m in METRICS} for p in portfolios}
    for _ in range(runs):
        world = draw_world(prm, rng)
        for p in portfolios:
            r = eval_portfolio(prm, p, world)
            for m in METRICS:
                acc[p['id']][m].append(r[m])
    return acc

def summarize(acc):
    out = {}
    for pid, d in acc.items():
        out[pid] = {}
        for m, xs in d.items():
            mu = statistics.fmean(xs)
            sd = statistics.pstdev(xs) if len(xs) > 1 else 0.0
            hw = 1.96 * sd / math.sqrt(len(xs))
            out[pid][m] = (mu, hw)
    return out

# ---------------------------------------------------------------- sanity checks

def zero_hazards(prm):
    p = copy.deepcopy(prm)
    for c in p['caps'].values():
        c['hazard'] = 0.0; c['removal'] = 0.0
    p['shock']['A'] = p['shock']['O'] = 0.0
    return p

def check_zero(prm, runs=300):
    p = zero_hazards(prm)
    acc = summarize(simulate(p, runs, 11))
    bad = []
    for pf in PORTFOLIOS:
        unavailable = any(p['caps'][c].get('unavailable_at_start') for c in pf['caps'])
        r = acc[pf['id']]
        if unavailable:
            # with zero hazards it may still arrive by improvement; require it to fail at least at month 1
            continue
        full0 = 1.0 if pf['capture']({c: OK for c in pf['caps']}) == 'full' else 0.0
        if not (r['cap_works'][0] == 1 and r['inj_works'][0] == 1 and r['inj_clean'][0] == 1
                and r['cap_full'][0] == full0 and r['silent_any'][0] == 0):
            bad.append(pf['id'])
    # options resting on a capability absent at the start must not work in month 1
    p2 = copy.deepcopy(p); p2['improve']['unavailable'] = 0.0
    acc2 = summarize(simulate(p2, 50, 12))
    for pf in PORTFOLIOS:
        if any(p2['caps'][c].get('unavailable_at_start') for c in pf['caps']):
            if acc2[pf['id']]['cap_works'][0] != 0:
                bad.append(pf['id'] + '(unavailable cap works)')
    return bad

def check_removal(prm, cap, runs=200):
    p = zero_hazards(prm)
    p['caps'][cap]['removal'] = 1.0
    p['improve']['replacement'] = 0.0
    p['vendor_fix']['regression_share'] = 0.0
    p['caps'][cap].pop('unavailable_at_start', None)
    rng = random.Random(21)
    problems = []
    for _ in range(runs):
        world = draw_world(p, rng)
        for pf in PORTFOLIOS:
            if any(p['caps'][c].get('unavailable_at_start') for c in pf['caps']):
                continue
            # month-1 sample only, then the whole horizon
            r = eval_portfolio(p, pf, world)
            touches = cap in pf['caps'] and pf['host'] in p['caps'][cap]['on']
            if not touches:
                if not (r['cap_works'] == 1 and r['inj_works'] == 1):
                    problems.append((pf['id'], 'affected though it does not depend on ' + cap))
            else:
                # does losing this capability alone defeat capture or injection?
                s = {c: OK for c in pf['caps']}; s[cap] = ABSENT
                lc, li = pf['capture'](s), pf['inject'](s)
                fails_c, fails_i = lc in ('none', 'wrong'), li in ('none', 'lost')
                if fails_c and r['cap_works'] != 0:
                    problems.append((pf['id'], 'capture works in some month though ' + cap + ' is removed'))
                if fails_i and r['inj_works'] != 0:
                    problems.append((pf['id'], 'injection works in some month though ' + cap + ' is removed'))
    return sorted(set(problems))

# ---------------------------------------------------------------- main

def main():
    args = sys.argv[1:]
    prm = json.load(open(args[0])); outdir = args[1]
    os.makedirs(outdir, exist_ok=True)
    runs = prm['runs']
    if '--runs' in args:
        runs = int(args[args.index('--runs') + 1])
    t0 = time.time()
    report = {'sanity': {}}

    z = check_zero(prm)
    report['sanity']['zero_hazards'] = {'failing_portfolios': z, 'passed': not z}
    print('sanity zero hazards:', 'PASS' if not z else 'FAIL ' + str(z))
    rem = {}
    for cap in prm['caps']:
        pr = check_removal(prm, cap)
        rem[cap] = pr
    allok = all(not v for v in rem.values())
    report['sanity']['removal_one_cap'] = {'problems': {k: v for k, v in rem.items() if v}, 'passed': allok}
    print('sanity removal of each capability:', 'PASS' if allok else 'FAIL ' + json.dumps({k: v for k, v in rem.items() if v})[:2000])
    if not (allok and not z):
        json.dump(report, open(os.path.join(outdir, 'results.json'), 'w'), indent=1)
        sys.exit('sanity check failed; stopping')

    # seed stability: three seeds at a quarter of the runs each
    seeds = {}
    for sd in (101, 202, 303):
        seeds[sd] = summarize(simulate(prm, max(runs // 4, 1000), sd))
    maxdiff = {}
    for m in METRICS:
        maxdiff[m] = max(abs(seeds[a][pid][m][0] - seeds[b][pid][m][0])
                         for pid in seeds[101] for a in seeds for b in seeds)
    report['sanity']['seed_stability'] = {'runs_each': max(runs // 4, 1000), 'max_abs_diff': maxdiff}
    print('seed stability (max abs diff across 3 seeds):', {k: round(v, 4) for k, v in maxdiff.items()})
    if '--checks-only' in args:
        json.dump(report, open(os.path.join(outdir, 'results.json'), 'w'), indent=1)
        return

    acc = simulate(prm, runs, prm['seed'])
    summ = summarize(acc)
    report['runs'] = runs
    report['results'] = {pid: {m: {'mean': v[0], 'ci95': v[1]} for m, v in d.items()} for pid, d in summ.items()}
    report['portfolios'] = {p['id']: {k: p[k] for k in ('host', 'name', 'build', 'status', 'values')} for p in PORTFOLIOS}
    # paired differences against the current option on the same host (common random numbers)
    cur = {'cc_local': 'CC1', 'cc_cloud': 'CL1', 'cowork_local': 'CW1', 'cowork_cloud': 'KC1', 'codex_local': 'CX1', 'codex_cloud': 'XC1'}
    diffs = {}
    for p in PORTFOLIOS:
        base = cur[p['host']]
        if base == p['id']: continue
        diffs[p['id']] = {}
        for m in METRICS:
            d = [x - y for x, y in zip(acc[p['id']][m], acc[base][m])]
            mu = statistics.fmean(d); hw = 1.96 * statistics.pstdev(d) / math.sqrt(len(d))
            diffs[p['id']][m] = {'vs': base, 'mean': mu, 'ci95': hw}
    report['paired_diffs'] = diffs
    report['seconds'] = round(time.time() - t0, 1)
    json.dump(report, open(os.path.join(outdir, 'results.json'), 'w'), indent=1)
    with open(os.path.join(outdir, 'results.csv'), 'w') as f:
        f.write('id,host,' + ','.join(m + ',' + m + '_ci95' for m in METRICS) + '\n')
        for p in PORTFOLIOS:
            r = summ[p['id']]
            f.write(p['id'] + ',' + p['host'] + ',' + ','.join('%.4f,%.4f' % r[m] for m in METRICS) + '\n')
    print('done in', report['seconds'], 's')

if __name__ == '__main__':
    main()
