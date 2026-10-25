//go:build !olderdataversion

package main

// dataVersion is the version of the data this program writes in the state directory,
// and the highest it reads (D-244, the data in the state directory carries one
// version). Raise it by one in a release whose writes an older program would misread
// or damage, such as a column an older program does not know or a field it drops when
// it rewrites a file, and in no other release, because every raise leaves each older
// program still installed unable to touch the machine's data. A raise needs nothing
// else: an existing state directory is brought up to it by upgradeStateDirectory,
// which runs every migration before it writes the number.
//
// Version 1 is the user identity: identity.json carries the user and the chain, and
// membership.db and every room database have the columns it adds.
const dataVersion = 1
