"""One-at-a-time sensitivity of the simulation's results to the parameters least
supported by the history. Run: python3 -I sens.py params.json OUTDIR [runs]
Imports sim.py from its own directory; writes sensitivity.json and prints a table."""
import json, sys, os, copy
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import sim

prm = json.load(open(sys.argv[1])); outdir = sys.argv[2]
runs = int(sys.argv[3]) if len(sys.argv) > 3 else 2500
KEEP = sys.argv[4].split(',') if len(sys.argv) > 4 else ['CC1', 'CC3', 'CC6', 'CL1', 'CL2', 'CW1', 'CW2', 'KC1', 'KC2', 'CX1', 'CX3']
PF = [p for p in sim.PORTFOLIOS if p['id'] in KEEP]

def setcap(cap, v):
    def f(p): p['caps'][cap]['hazard'] = v
    return f
def setdet(k, v):
    def f(p): p['detect'][k] = v
    return f
def shocks(x):
    def f(p): p['shock']['A'] *= x; p['shock']['O'] *= x
    return f
def allhaz(x):
    def f(p):
        for c in p['caps'].values(): c['hazard'] = min(1.0, c['hazard'] * x); c['removal'] = min(1.0, c['removal'] * x)
        p['shock']['A'] *= x; p['shock']['O'] *= x
    return f

CASES = [
    ('base', lambda p: None),
    ('transcript hazard 0.02', setcap('A_transcript', 0.02)),
    ('transcript hazard 0.15', setcap('A_transcript', 0.15)),
    ('rollout hazard 0.03', setcap('O_rollout', 0.03)),
    ('rollout hazard 0.30', setcap('O_rollout', 0.30)),
    ('MessageDisplay hazard 0.12', setcap('A_msgdisp', 0.12)),
    ('cloud environment hazard 0.03', setcap('A_cloudenv', 0.03)),
    ('cloud environment hazard 0.20', setcap('A_cloudenv', 0.20)),
    ('users notice silence in 0.5 mo', setdet('user_silent_mean', 0.5)),
    ('users notice silence in 4 mo', setdet('user_silent_mean', 4.0)),
    ('doctor lag 1 mo', setdet('doctor_mean', 1.0)),
    ('guard catches 50%', setdet('guard_catch', 0.5)),
    ('shocks x0.25', shocks(0.25)),
    ('shocks x4', shocks(4.0)),
    ('all hazards x0.5', allhaz(0.5)),
    ('all hazards x2', allhaz(2.0)),
]
out = {}
for name, f in CASES:
    p = copy.deepcopy(prm); f(p)
    s = sim.summarize(sim.simulate(p, runs, 7, PF))
    out[name] = {pid: {m: s[pid][m] for m in ('cap_works', 'inj_works', 'silent_any', 'cost')} for pid in s}
    print(name.ljust(34), ' '.join('%s %.3f/%.3f/%.2f' % (pid, out[name][pid]['cap_works'][0], out[name][pid]['inj_works'][0],
                                                     out[name][pid]['silent_any'][0]) for pid in KEEP), flush=True)
json.dump({'runs': runs, 'cases': out}, open(os.path.join(outdir, 'sensitivity.json'), 'w'), indent=1)
