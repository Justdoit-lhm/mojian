# Releasing Mojian

A release is a tag. Pushing `vX.Y.Z` runs the Release workflow
(`.github/workflows/release.yml`), which tests the tree on macOS, Linux
and Windows, builds the macOS app for Apple Silicon and Intel, the Linux
archives and packages for amd64 and arm64, and the Windows portable zip,
then publishes the GitHub release with every installer.

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
`mojian-X.Y.Z-windows-amd64.zip`, `install.sh` and the matching `.deb`
packages, creates a draft release with generated notes, uploads every
installer and `SHA256SUMS`, and publishes the draft only after every upload
succeeds. If a job fails, fix it, move the tag to
the new commit and push the tag again; the workflow replaces the draft.
A published release is not overwritten.

The app is ad-hoc signed because this repository has no Developer ID
certificate or notary credentials. A downloaded build may need
right-click → Open on first launch. Add `macos.signingIdentity` and
`macos.notarize` to `mygo.json` and the matching macOS secrets to the
workflow before shipping a notarized build.
