"""Prints the space each table and index of a room database takes, and the bytes each column of its
events holds on average. Run with `python3 -I storesize.py <copy of a COGMER_HOME>`, on a copy, never on a
daemon's own files, since it opens the first database under rooms/."""
import sqlite3, sys, glob
db = sqlite3.connect(glob.glob(sys.argv[1] + "/rooms/*.db")[0])
try:
    rows = db.execute("SELECT name, SUM(pgsize), COUNT(*) FROM dbstat GROUP BY name ORDER BY 2 DESC").fetchall()
    print("space by table or index (bytes, pages):")
    for r in rows: print("  ", r)
except Exception as e:
    print("no dbstat:", e)
for (t,) in db.execute("SELECT name FROM sqlite_master WHERE type='table'"):
    print(t, db.execute("SELECT COUNT(*) FROM %s" % t).fetchone()[0])
cols = [r[1] for r in db.execute("PRAGMA table_info(events)")]
n = db.execute("SELECT COUNT(*) FROM events").fetchone()[0]
print("events:", n)
for c in cols:
    s = db.execute("SELECT SUM(LENGTH(CAST(%s AS BLOB))) FROM events" % c).fetchone()[0] or 0
    print("  %-18s %8.0f bytes per event" % (c, s / n))
print(db.execute("SELECT event_type, COUNT(*), AVG(LENGTH(content)), AVG(LENGTH(metadata)) FROM events GROUP BY event_type").fetchall())
print(db.execute("PRAGMA page_size").fetchone(), db.execute("PRAGMA freelist_count").fetchone())
