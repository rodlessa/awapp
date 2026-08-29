# Build awapp into a .deb and .rpm (and a tarball) from the current tree.
# Needs: go, tar, gzip, dpkg-deb, rpmbuild (optional).
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="${1:-1.0.5}"
STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT

# --- Build the release binary ---
go build -trimpath -ldflags="-s -w -X main.version=v${VERSION}" -o "$STAGE/awapp" .

# --- .deb ---
if command -v dpkg-deb >/dev/null 2>&1; then
  mkdir -p "$STAGE/deb/usr/bin" "$STAGE/deb/usr/share/man/man1" "$STAGE/deb/DEBIAN"
  install -m755 "$STAGE/awapp" "$STAGE/deb/usr/bin/awapp"
  install -m644 packaging/man/awapp.1 "$STAGE/deb/usr/share/man/man1/awapp.1"
  sed "s/^Version: .*/Version: ${VERSION}/" packaging/deb/control > "$STAGE/deb/DEBIAN/control"
  (cd "$STAGE" && dpkg-deb --build deb "awapp_${VERSION}-1_amd64.deb" >/dev/null)
  cp "$STAGE/awapp_${VERSION}-1_amd64.deb" packaging/
else
  echo "dpkg-deb not found — skipping .deb"
fi

# --- .rpm (needs rpmbuild) ---
if command -v rpmbuild >/dev/null 2>&1; then
  mkdir -p "$STAGE/rpmbuild"/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
  tar -czf "$STAGE/awapp-${VERSION}.tar.gz" .
  cp "$STAGE/awapp-${VERSION}.tar.gz" "$STAGE/rpmbuild/SOURCES/"
  rpmbuild --define "_topdir $STAGE/rpmbuild" -bb packaging/rpm/awapp.spec \
    --define "_version ${VERSION}"
  cp "$STAGE/rpmbuild/RPMS/x86_64/"awapp-*.rpm packaging/ 2>/dev/null || true
else
  echo "rpmbuild not found — skipping .rpm (deb built at packaging/awapp_${VERSION}-1_amd64.deb)"
fi

# --- plain tarball ---
mkdir -p "$STAGE/tar/usr/bin" "$STAGE/tar/usr/share/man/man1"
install -m755 "$STAGE/awapp" "$STAGE/tar/usr/bin/awapp"
install -m644 packaging/man/awapp.1 "$STAGE/tar/usr/share/man/man1/awapp.1"
tar -C "$STAGE/tar" -czf "packaging/awapp-${VERSION}-linux-amd64.tar.gz" .

echo "done → packaging/awapp_${VERSION}-1_amd64.deb (+ .rpm / .tar.gz if available)"
