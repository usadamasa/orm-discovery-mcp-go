---
on:
  schedule: weekly

permissions:
  contents: read
  issues: read
  pull-requests: read

safe-outputs:
  create-issue:
    title-prefix: "[repo status] "
    labels: [report]
  update-issue:
    body:
    status:
    target: "*"

tools:
  github:
---

# Weekly Repo Status Report

Before creating or updating the report, follow these steps:

1. Search for existing open issues with the label "report" in this repository
2. Filter to issues whose title starts with "[repo status]"
3. Based on the result:

   **Case A - Existing issue found:**
   - Generate the new weekly report content
   - Compare it with the existing issue body
   - If changes are MINOR (less than 50% of content changed): update the existing issue body using the GitHub API
   - If changes are MAJOR (50% or more of content changed): close the old issue, then create a new issue using the create-issue tool

   **Case B - No existing issue found:**
   - Create a new issue using the create-issue tool

## Report contents

Include:

- Recent repository activity (issues, PRs, discussions, releases, code changes)
- Progress tracking, goal reminders and highlights
- Project status and recommendations
- Actionable next steps for maintainers

Keep it concise and link to the relevant issues/PRs.
