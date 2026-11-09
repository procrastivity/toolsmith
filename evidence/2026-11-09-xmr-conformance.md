# xmr conformance run — closeout, 2026-11-09

- **Command**: `toolsmith check /home/bsimensen/Code/xmr` (toolsmith
  `v0.0.3-2-g92fcfcf`)
- **Tip**: `70d94f6` on branch `main` (xmr's reconcile to contract v1.6;
  the repository is private)
- **Result**: exit 1, two findings:

  ```
  C6.5: no .github/workflows/ci.yml
  C6.5: no .github/workflows/release.yml
  toolsmith: check: 2 finding(s) for /home/bsimensen/Code/xmr
  ```

- **Context**: run as the seal condition of xmr's closeout. xmr runs
  GitLab CI (`.gitlab-ci.yml`), so the two C6.5 findings are the known
  standing pair, the same as ste9's, pending the GitLab-equivalence
  draft ([feedback/xmr.md](../feedback/xmr.md) item 9). No C3.10
  finding: the manifest records `contractReconciledMinor` 6. The
  conversion was evidence-only: no parity gate and no oracle, so there
  is no parity-final record. `make check` was green on the same tip.
