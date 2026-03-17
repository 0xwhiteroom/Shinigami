<div align="center">

```
  ███████╗██╗  ██╗██╗███╗   ██╗██╗ ██████╗  █████╗ ███╗   ███╗██╗
  ██╔════╝██║  ██║██║████╗  ██║██║██╔════╝ ██╔══██╗████╗ ████║██║
  ███████╗███████║██║██╔██╗ ██║██║██║  ███╗███████║██╔████╔██║██║
  ╚════██║██╔══██║██║██║╚██╗██║██║██║   ██║██╔══██║██║╚██╔╝██║██║
  ███████║██║  ██║██║██║ ╚████║██║╚██████╔╝██║  ██║██║ ╚═╝ ██║██║
  ╚══════╝╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝╚═╝ ╚═════╝╚═╝  ╚═╝╚═╝     ╚═╝╚═╝
```

# shinigami 死神

### *Smart Spider & Directory Hunter*

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Platform](https://img.shields.io/badge/Platform-Linux%20amd64-lightgrey?style=flat-square&logo=linux)](.)
[![Version](https://img.shields.io/badge/Version-2.0.0-blueviolet?style=flat-square)](.)
[![0xAscension](https://img.shields.io/badge/0xAscension-red?style=flat-square)](https://github.com/0xAscension)

> **No wordlist needed.** Shinigami crawls a target's own HTML, JavaScript, comments, and forms to intelligently discover hidden paths — then mutates what it finds to go deeper.

</div>

---

##  Why shinigami?

| Feature |  |  | **shinigami** |
|---------|--------|-------------|--------------|
| No wordlist needed |  |  | ✅ |
| JS crawl | |  | ✅ |
| Comment mining |  |  | ✅ |
| Path mutation |  |  | ✅ |
| WAF detection |  |  | ✅ |
| Smart filter |  |  | ✅ |
| Juicy path highlight |  |  | ✅ |
| Resume scan |  |  | ✅ |
| JSONL output |  |  | ✅ |

---

##  Features

-  **Smart Crawl** — extracts paths from HTML, JS, comments, forms, API references
-  **Path Mutation** — auto-generates smart variants of discovered paths
-  **WAF Detection** — detects 15+ WAFs before crawling begins
-  **robots.txt / sitemap** — auto-fetches and parses both
-  **Juicy Highlighting** — marks high-value paths (admin, login, config, backup, debug...)
-  **Smart Filter** — skips noisy WordPress/asset/theme paths
-  **Resume Scan** — save and resume interrupted scans
-  **Rate Limiting** — built-in req/s throttle to avoid detection
-  **Scope Control** — include/exclude regex patterns
-  **Output Formats** — JSON · TXT · JSONL (katana-compatible)

---

##  Flags

```
INPUT
  -u <url>             Target URL  (required)

SCAN MODES
  -smart               Skip noisy paths (wp-content, assets, themes, plugins)
  -juicy               Highlight high-value paths (admin, login, config, backup...)

CONFIG
  -t <int>             Threads                        (default: 20)
  -depth <int>         Max crawl depth                (default: 3)
  -rl <int>            Rate limit req/s               (default: 0 = unlimited)
  -timeout <sec>       Request timeout seconds        (default: 10)
  -proxy <url>         Proxy URL (http:// or socks5://)
  -H <name:value>      Custom header (repeatable)
  -ua <string>         Custom User-Agent
  -sub                 Include subdomains in scope
  -no-mutate           Disable path mutation engine

FILTER
  -include <regex>     Only crawl paths matching regex
  -exclude <regex>     Skip paths matching regex
  -all                 Show all results including 404s

OUTPUT
  -o <file>            Save results as JSON
  -ot <file>           Save results as TXT
  -ojsonl <file>       Save results as JSONL (katana-compatible)
  -resume <file>       Save and resume scan state

OTHER
  -firewall            WAF detection only — no crawl
  --install-license    Activate license on this machine
```

---

##  Examples

```bash
# Basic crawl
shinigami -u https://target.com

# Smart mode — skip noise, highlight juicy paths
shinigami -u https://target.com -smart -juicy

# Deep crawl with rate limiting
shinigami -u https://target.com -depth 5 -rl 20 -t 30

# Through Burp proxy
shinigami -u https://target.com -proxy http://127.0.0.1:8080

# Custom auth header
shinigami -u https://target.com -H 'Cookie: session=abc123'

# Include subdomains
shinigami -u https://target.com -sub

# Scope to specific path
shinigami -u https://target.com -include '\.php$'

# Exclude logout/static
shinigami -u https://target.com -exclude '/logout|/static'

# Save all output formats
shinigami -u https://target.com -smart -juicy -o result.json -ot result.txt

# WAF check only (no crawl)
shinigami -u https://target.com -firewall

# Save and resume
shinigami -u https://target.com -resume scan.json
```

---

##  Output

```
  STATUS   SOURCE   DEPTH   PATH                                              SIZE
  ─────────────────────────────────────────────────────────────────────────────────
  200      [BOT]    0       /robots.txt                                       45b
  200      [MAP]    0       /sitemap.xml                                      2.1KB
  200      [HTML]   0       /                                                 24KB
  200      [JS]     1       /api/v1/users                                     1.2KB   ◄ JUICY
  403      [MUT]    2       /admin/dashboard                                  0b      ◄ JUICY
  302      [HTML]   1       /login                                            0b      ◄ JUICY
  200      [CMT]    2       /debug/console                                    8.3KB   ◄ JUICY
  200      [FRM]    1       /upload                                           4.2KB   ◄ JUICY
```

### Source Tags

| Tag | Source |
|-----|--------|
| `[HTML]` | Extracted from HTML links |
| `[JS]` | Extracted from JavaScript |
| `[CMT]` | Found in code comments |
| `[API]` | API endpoint references |
| `[FRM]` | Form action paths |
| `[MUT]` | Path mutation results |
| `[BOT]` | From robots.txt |
| `[MAP]` | From sitemap.xml |

---

##  Installation

```bash
# Build
unzip shinigami-v2.zip -d shinigami && cd shinigami
bash build.sh

# Install
sudo dpkg -i shinigami_2.0.0_amd64.deb

# Verify
shinigami --version
```

> **Requirements:** Go 1.21+ · Linux amd64

---

##  Disclaimer

> For authorized security testing and educational purposes only.
> Use only on systems you have explicit permission to test.

---

<div align="center">

*shinigami 死神 v1.0 — 0xWHITEROOM  「サイバー守護者」*

**[0xWHITEROOM](https://github.com/0xAscension)** · *We don't hack systems. We ascend them.*

</div>
