# Asteriade — Codex repository instructions

## Identity and source of truth

- The product name is **Asteriade**; do not rename it to a generic multi-agent tool.
- Treat `docs/REQUIREMENTS.md` (v1.1 or its approved successor) as the requirements source of truth.
- Do not silently change requirements. Record ambiguities and request a decision only when it materially blocks safe progress.

## Engineering constraints

- Prioritize Windows 11 and a standalone `asteriade.exe` for the first release. Node.js, npm, Docker, and an external database must not be required **at runtime**.
- Keep the Control Plane, Orchestrator, Runner, UI, and storage boundaries separable for future EC2 and mobile access.
- Implement and validate enforceable permissions and sandbox boundaries **before** autonomous tool execution.
- Fail closed when permissions, external-service behavior, model capabilities, costs, or execution state are unknown.
- Never treat a model's claim of completion as sufficient evidence: verify required artifacts and tests.
- Do not bypass third-party product terms, sandbox controls, approval gates, or spending limits.

## Subagent delegation (Codex development workflow, not product requirements)

- Prefer **GPT-6-Luna (`gpt-6-luna`)** for bounded, independent tasks when it is available in the current Codex session.
- Good Luna tasks: codebase exploration, documentation research, requirements cross-checks, test-case design, log analysis, small independent edits, and focused test implementation.
- Delegate proactively when doing so improves speed or quality; for trivial or strictly sequential work, use the parent agent only.
- Do not delegate major architecture decisions, final security approvals, release decisions, or final integration accountability. The parent agent owns those decisions and validates subagent results.
- Give each subagent a narrow scope, expected deliverable, and relevant file paths. Wait for results and synthesize evidence before finalizing.
- Do not let parallel agents edit the same files or shared mutable state without explicit coordination. Prefer read-only parallel tasks first.
- Keep simultaneous subagents within the configured limit (initially 3); do not spawn agents merely to increase the count.
- If GPT-6-Luna or subagents are unavailable, use an available permitted alternative or work directly. Do not repeatedly retry an unsupported model, claim Luna was used when it was not, or initiate separately billed API use without user approval.
- Report which subtasks were delegated, which models actually executed them (if observable), and any fallback or inability to delegate.

## Workflow and approvals

- For substantial work, first propose the scope, acceptance tests, risks, and a stage-by-stage plan. Execute only the approved stage.
- When explicitly asked for design documents only, do not modify application code, dependencies, CI, or deployments.
- Require user approval for git push, releases, deployments, external posting/sending, purchases, new metered API spend, destructive changes, or broad permission changes.
- Prefer minimal changes, meaningful automated tests, and observed evidence. State clearly what was not tested.
- Preserve a concise handoff at the end of each approved stage: files changed, tests run, results, remaining risks, and next proposed stage.
