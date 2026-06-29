# Branch Protection Policy

Target branch: `main`

Required settings for CAL-009:

- Require a pull request before merging.
- Require at least 1 approving review.
- Require review from Code Owners.
- Dismiss stale approvals when new commits are pushed.
- Require conversation resolution before merging.
- Require status checks to pass before merging.
- Require branches to be up to date before merging.
- Required status checks:
  - `Secrets (gitleaks)`
  - `Backend (lint · proto · test · coverage · sonar)`
  - `Frontend (typecheck · build · lint · test)`
  - `Supply chain (govulncheck · npm audit · Trivy)`
- Block force pushes.
- Block branch deletion.
- Do not allow direct pushes to `main`.

SonarQube/SonarCloud remains part of the backend job until the SonarCloud project
and `SONAR_TOKEN` secret are available.
