# Zync

Simple two-node file syncing service.

## Use

There are two nodes involved in the exchange: one local, and one remote. The
remote node is started with the command `zync -d`. Port and proxy can be
specified, as well as the root path to sync (by default, the current working
directory).

The local node is started with the command `zync -c {remote node}`. By default,
Zync runs in non-destructive, non-interactive mode (see options below); this
will copy files _from_ the local node _to_ the remote node without overwriting
any files on the remote node. Warnings will be issued for any conflicts, but
they will not be changed on either node.

## Options

**`--daemon, -d`** 
Runs the node in daemon mode. Non-interactive only.

**`--port {number}, -p {number}`** 
For daemon mode only; serves at the specified port. By default, the port 20741
is used.

**`--connect {remote}, -c {remote}`** 
Connects to the remote node, specified as a URI. Supported schemes include
`zync://`, which connects to a Zync daemon, and `file://`, which operates
on the local (or network attached) file system.

**`--keep {mine|theirs}, -k {mine|theirs}`** 
If a conflict occurs, keep 'mine' (the local node) or 'theirs' (the remote
node).

**`--interactive, -i`** 
Run in interactive mode; any time a conflict occurs, ask the user what to do.
Options are `m` (mine), `t` (theirs), or `s` (skip).

**`--delete`** 
Deletes all files on the remote node that no longer exist on the local node.

**`--reverse, -r`** 
Reverses the roles of the nodes, making the local (`zync -c`) node act as the
remote node and vice versa for all file resolution purposes. Any command output
and interactive prompts will still occur at the local node.

**`--verbose, -v`** 
Enables verbose logging. All file events will be output, even when no changes
were made.

**`--hash, -h`** 
Computes a checksum of potentially conflicting files rather than relying on the
file size.
