# Releasing Mojian

A release is a tag. Pushing `vX.Y.Z` runs the Release workflow
(`.github/workflows/release.yml`), which tests the tree on macOS, Linux
and Windows, builds the macOS app for Apple Silicon and Intel, the Linux
archives and packages for amd64 and arm64, and the Windows installers for
amd64 and arm64, then publishes the GitHub release with every installer.
Windows has been a release target since 0.3.0.

## Steps

1. Pick the version: a patch (`0.1.1` → `0.1.2`) for fixes, a minor for
   features.
2. Set it in `mygo.json` (`"version"`).
3. Add its section to the top of `CHANGELOG.md`.
4. Commit them and tag the commit:

   ```sh
   git add mygo.json CHANGELOG.md
   git commit -m "Release X.Y.Z"
   git tag vX.Y.Z
   git push origin main
   git push origin vX.Y.Z
   ```

5. Wait for the Release run of the tag:

   ```sh
   gh run list --workflow release.yml --limit 1
   gh run watch <run id> --exit-status
   ```

The workflow refuses a tag that does not match `mygo.json`, builds
`mojian-X.Y.Z-arm64.dmg`, `mojian-X.Y.Z-amd64.dmg`,
`mojian-X.Y.Z-linux-amd64.tar.gz`, `mojian-X.Y.Z-linux-arm64.tar.gz`,
`mojian-setup-X.Y.Z-amd64.exe`, `mojian-setup-X.Y.Z-arm64.exe`,
the signed `*-darwin-*.tar.gz`, `*-linux-*.tar.gz` and
`*-windows-*.tar.gz` update archives, `update-<target>.json` manifests,
`install.sh`, `install.ps1` and the matching `.deb` packages, creates a
draft release with generated notes, uploads every installer and
`SHA256SUMS`, and publishes it as the latest release only after every
upload succeeds.

Updates are signed with the Ed25519 key pair from `go tool mygo keygen`.
`mygo.json` holds only the public key; set the private key as the
`MYGO_UPDATER_PRIVATE_KEY` GitHub Actions secret. Never commit
`mygo-update.key`; losing it prevents installed apps from updating.

If a job fails, fix it, move the tag to the new commit and push the tag
again; the workflow replaces the draft. A published release is not
overwritten.

The app is ad-hoc signed because this repository has no Developer ID
certificate or notary credentials. A downloaded build may need
right-click → Open on first launch. Add `macos.signingIdentity` and
`macos.notarize` to `mygo.json` and the matching macOS secrets to the
workflow before shipping a notarized build.

Windows builds are unsigned for now: SmartScreen may warn on the
installer and the executable. To sign them, set `windows.certificate`
or `windows.signCommand` in `mygo.json` and provide
`MYGO_WINDOWS_CERTIFICATE_PASSWORD` or the signing tool in the Windows
job. The Windows job builds NSIS installers on the Windows runner; NSIS
is downloaded when the runner does not have it.
