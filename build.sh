#!/bin/bash
# SHINIGAMI 死神 v2.0 — Build Script  by FIN
set -e

RED='\033[91m'; GREEN='\033[92m'; YELLOW='\033[93m'
CYAN='\033[96m'; BOLD='\033[1m'; RST='\033[0m'

echo ""
echo -e "${RED}  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░${RST}"
echo -e "${RED}  ░  ${BOLD}SHINIGAMI 死神 v1.0 — Build Script${RST}${RED}         ░${RST}"
echo -e "${RED}  ░  ${YELLOW}by 0xWHITEROOM  「0xホワイトルーム」${RST}${RED}                    ░${RST}"
echo -e "${RED}  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░${RST}"
echo ""

# Check Go
if ! command -v go &>/dev/null; then
    echo -e "${RED}[-]${RST} Go not installed!"
    echo ""
    echo "    wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz"
    echo "    sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz"
    echo "    export PATH=\$PATH:/usr/local/go/bin"
    echo "    echo 'export PATH=\$PATH:/usr/local/go/bin' >> ~/.bashrc"
    exit 1
fi
echo -e "${GREEN}[+]${RST} Go: $(go version)"

# Deps
echo -e "${CYAN}[*]${RST} go mod tidy..."
go mod tidy

# Build
echo -e "${CYAN}[*]${RST} Building shinigami..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -trimpath \
    -o shinigami ./cmd/shinigami/

[ -f shinigami ] || { echo -e "${RED}[-]${RST} Build failed!"; exit 1; }
SIZE=$(ls -lh shinigami | awk '{print $5}')
echo -e "${GREEN}[+]${RST} Binary: ${BOLD}${SIZE}${RST} — $(file shinigami | cut -d',' -f1-2)"

# .deb packaging
echo ""
echo -e "${CYAN}[*]${RST} Packaging .deb..."

DEB="deb/shinigami"
mkdir -p "${DEB}/DEBIAN"
mkdir -p "${DEB}/usr/local/bin"
mkdir -p "${DEB}/usr/share/doc/shinigami"

cp shinigami "${DEB}/usr/local/bin/shinigami"
chmod 755 "${DEB}/usr/local/bin/shinigami"

cat > "${DEB}/DEBIAN/control" << 'CTRL'
Package: shinigami
Version: 2.0.0
Architecture: amd64
Maintainer: FIN <fin@protonmail.com>
Description: Shinigami 死神 v2.0 — Smart Spider & Directory Hunter
 No wordlist needed. Intelligently crawls HTML, JS, comments, forms.
 Features: Rate limiting, Scope control, Resume scan, JSONL output,
 Proxy support, Path mutation engine, Redirect following.
 by FIN 「サイバー守護者」
Depends:
Priority: optional
Section: net
Installed-Size: 6144
CTRL

cat > "${DEB}/DEBIAN/postinst" << 'POST'
#!/bin/bash
echo ""
echo "╔═══════════════════════════════════════════════╗"
echo "║   死神 SHINIGAMI v1.0 installed!              ║"
echo "║   shinigami -u https://example.com            ║"
echo "╚═══════════════════════════════════════════════╝"
echo ""
POST
chmod 755 "${DEB}/DEBIAN/postinst"

cat > "${DEB}/DEBIAN/prerm" << 'PRERM'
#!/bin/bash
echo "Removing Shinigami 死神..."
PRERM
chmod 755 "${DEB}/DEBIAN/prerm"

cat > "${DEB}/usr/share/doc/shinigami/README" << 'README'
SHINIGAMI 死神 v2.0 — Smart Spider & Directory Hunter
by FIN  「サイバー守護者」

No wordlist needed!

BASIC USAGE:
  shinigami -u https://example.com

OPTIONS:
  -t 50               Threads
  -depth 5            Max depth
  -rl 10              Rate limit (10 req/s)
  -o out.json         JSON output
  -ot out.txt         TXT output
  -ojsonl out.jsonl   JSONL output (katana compatible)
  -resume scan.json   Save/resume scan
  -sub                Include subdomains
  -no-mutate          Disable mutation
  -proxy http://...   Proxy support
  -H 'Cookie: x=y'   Custom headers
  -include '\.php$'   Scope regex
  -exclude '/logout'  Exclude regex
  -all                Show all (incl 404)

INSTALL:
  sudo dpkg -i shinigami_2.0.0_amd64.deb

UNINSTALL:
  sudo dpkg -r shinigami
README

dpkg-deb --build "${DEB}" shinigami_2.0.0_amd64.deb
DEB_SIZE=$(ls -lh shinigami_2.0.0_amd64.deb | awk '{print $5}')
echo -e "${GREEN}${BOLD}[+]${RST} .deb: ${BOLD}shinigami_2.0.0_amd64.deb${RST} (${DEB_SIZE})"
echo ""
echo -e "${CYAN}[*]${RST} Install with:"
echo -e "    ${BOLD}sudo dpkg -i shinigami_2.0.0_amd64.deb${RST}"
echo ""
echo -e "${GREEN}${BOLD}  「死神の準備完了」 BUILD COMPLETE! 🌸${RST}"
echo ""
