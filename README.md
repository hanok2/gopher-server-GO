# Gophor

A Gopher server written in GoLang as a means of learning about the Gopher
protocol, and more GoLang.

Possibly beta quality soon? More likely alpha right now.

`build.sh` and you're set to go! Just put `gophor` somewhere in your path.

Linux only _for now_.

I'm unemployed (not due to lack of effort...) and work on open-source projects
like this and many others for free. If you would like to help support my work
that would be hugely appreciated 💕 https://liberapay.com/grufwub/

# Usage

```
gophor [args]
       -root           Change server root directory.
       -port           Change server listening port.
       -hostname       Change server hostname (FQDN, used to craft dir lists).
       -bind-addr      Change server bind-address (used in creating socket).
       -uid            Change UID to drop privileges to.
       -gid            Change GID to drop privileges to.
       -system-log     Path to gophor system log file, else use stderr.
       -access-log     Path to gophor access log file, else use stderr.
       -cache-check    Change file-cache freshness check frequency (in secs).
       -cache-size     Change max no. files in file-cache.
       -cache-file-max Change maximum allowed size of a cached file.
       -page-width     Change page width used when formatting output.
       -gc-ballast     Change garbage-collector ballast size, default off.
```

# Features

- Built with concurrency and efficiency in mind.

- ZERO external dependencies.

- Security focused -- chroots into server direrctory and drops
  privileges. `maybe wait until stable release before use outside of hobby
  setups.`

- LRU file caching -- with user-controlled cache size, max cached file size
  and cache refresh frequency.

- Insert files within gophermaps, including automating reflowing of lines
  longer than (user definable) page width.

- Automatic replacement of `$hostname` with your server's selected hostname
  if you put `$hostname` in your gophermap... Wanna hear me say hostname one
  more time?

- Item type characters beyond RFC 1436 standard (see below).

- Separate system and access logging with output to file if requested (or to
  disable both).

- Support for a garbage-collector ballast of user-defined size, which works
  to reduce the frequency of garbage collection (see here).*

* Realistically, you're unlikely to run into issues anywhere even close to
  the scale mentioned in the Twitch blog post. But in a situation where
  you're running this on a low-powered device and hitting CPU constraints
  when many users are accessing resources, this _could_ help, maybe.
  Ultimately, I just read the blog post and really liked the idea so I
  implemented it.

# Supported gophermap item types

```
0 -- regular file (text)
1 -- directory (menu)
2 -- CSO phone-book server... should you be using this in 2020 lmao
3 -- Error
4 -- Binhexed macintosh file
5 -- DOS bin archive
6 -- Unix uuencoded file
7 -- Index-search server
8 -- Text-based telnet session
9 -- Binary file
T -- Text-based tn3270 session... in 2020???
g -- Gif format graphic
I -- Image file of some kind

+ -- Redundant server

. -- Lastline if this followed by CrLf

i -- Info message
h -- HTML document
s -- Audio file
p -- PNG image
d -- Document

M -- MIME type file
; -- Video file
c -- Calendar file
! -- Title
# -- Comment (not displayed)
- -- Hide file from directory listing
= -- Include subgophermap (prints file output here)
* -- Act as-if lastline and print directory listing below

Unavailable for now due to issues with accessing path within chroot:
$ -- Execute shell command and print stdout here
```

# Todos

- TLS support

- Connection throttling + timeouts

- Header + footer text

- Rotating logs

- Set default charset

- Autogenerated caps.txt

- Toggleable server status page (?)

- Proxy over HTTP support

- Finish inline shell scripting support

- Allow setting UID+GID via username string

- Fix file cache only updating if main gophermap changes (but not sub files)

- More fine-tuned handling of OS signals

- Add support for banned file extensions (i.e. not shown in dir listing)

# Please note

During the initial writing phase the quality of git commit messages may be
low and many changes are likely to be bundled together at a time, just
because the pace of development right now is rather break-neck.

As soon as we reach a stable point in development, or if other people start
contributing issues or PRs, whichever comes first, this will be changed
right away.

# Standards followed

Gopher-II (The Next Generation Gopher WWIS):
https://tools.ietf.org/html/draft-matavka-gopher-ii-00

All of the below can be viewed from your standard web browser using
floodgap's Gopher proxy:
https://gopher.floodgap.com/gopher/gw

RFC 1436 (The Internet Gopher Protocol:
gopher://gopher.floodgap.com:70/0/gopher/tech/rfc1436.txt

Gopher+ (upward compatible enhancements):
gopher://gopher.floodgap.com:70/0/gopher/tech/gopherplus.txt
