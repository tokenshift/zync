# Zync

Simple two-node file syncing service.

## Use

There are two nodes involved in the exchange: one local (the client), and one
remote (the server). The remote node is started with the command `zync -s`.
Port and proxy can be specified, as well as the root path to sync (by default,
the current working directory).

The local node is started with the command `zync -c {remote node}`. By default,
Zync runs in non-destructive, non-interactive mode (see options below); this
will copy files _from_ the local node _to_ the remote node without overwriting
any files on the remote node. Warnings will be issued for any conflicts, but
they will not be changed on either node.

## Options

**`--server, -s`** 
Runs the node in server mode. Non-interactive only.

**`--connect {remote}, -c {remote}`** 
Connects to the specified server.

**`--verbose, -v`** 
Enables verbose logging. All file events will be output, even when no changes
were made.

### Server Options

**`--port {number}, -p {number}`** 
Server will listen at the specified port. By default, the port 20741 is used.

**`--restrict, -r`**
The server will refuse to delete any of its own files, even if the client is
run with `-k mine --delete`. 

**`--Restrict, -R`**
The server will refuse to delete any of its own files OR overwrite them with
the client's version, even if the client is run with `-k mine --delete`.

### Client Options

**`--keep {mine|theirs}, -k {mine|theirs}`** 
If a conflict occurs, keep 'mine' (the local node) or 'theirs' (the remote
node).

**`--interactive, -i`** 
Run in interactive mode; any time a conflict occurs, ask the user what to do.
Options are `m` (mine), `t` (theirs), or `s` (skip).

**`--delete`** 
Deletes all files on the remote node that no longer exist on the local node.

**`--hash, -h`** 
Computes a checksum of potentially conflicting files rather than relying on the
file size.
