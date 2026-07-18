# HF Readmit Agent

> An evidence-first, human-in-the-loop research workstation for evaluating heart-failure post-discharge follow-up triage on **synthetic data**.

HF Readmit Agent is a working reader-study prototype. It lets a reviewer inspect a synthetic follow-up record, see the deterministic triage signals, document an independent decision, and evaluate agreement between rules, an optional LLM second reader, and reviewers.

**This is not a medical device or clinical decision-support system.** Do not use it for diagnosis, treatment, patient monitoring, or real patient data. Every record in this repository is synthetic.

## For hackathon judges — start here

### What you can verify in five minutes

1. Start the complete web, API, and PostgreSQL stack with one Docker command.
2. Review an 18-case synthetic cohort in a clinician-oriented workspace.
3. Inspect deterministic L0–L3 rules and the daily evidence that triggered them.
4. Record an independent reviewer decision with server-measured review time.
5. Explore aggregate agreement, safety, and export metrics.
6. Optionally connect an OpenAI-compatible LLM as a constrained, evidence-cited second reader and run a cohort Safety Lab benchmark.

The core reader-study workflow works **without an API key**. An LLM key is only needed for the optional research assistant and model benchmark.

### Fastest local start

**Prerequisites**

- Docker Desktop running with Compose v2
- Free local ports: 5173, 8080, and 5432

From the repository root:

~~~powershell
docker compose up -d --build
docker compose ps
~~~

Open:

| Surface | Address | What to look for |
| --- | --- | --- |
| Review application | http://localhost:5173 | Review Workspace, Settings, and Safety Lab |
| API health check | http://localhost:8080/healthz | status value is ok |
| PostgreSQL | localhost:5432 | Development-only persistence service |

Verify the API from PowerShell:

~~~powershell
(Invoke-RestMethod http://localhost:8080/healthz).status
~~~

Expected:

~~~text
ok
~~~

The first web startup can take longer because the development web container installs its locked Node dependencies. If it is not ready yet:

~~~powershell
docker compose logs -f web api
~~~

Stop the stack when finished:

~~~powershell
docker compose down
~~~

This preserves local PostgreSQL review decisions in Docker volumes. To intentionally delete all local study decisions and installed web dependencies:

~~~powershell
docker compose down -v
~~~

## The problem this prototype explores

Post-discharge heart-failure follow-up produces small, longitudinal signals: weight changes, oxygenation, symptoms, medication adherence, and patient-reported concerns. A simple alert alone is not enough for a rigorous evaluation workflow. Reviewers need to see the evidence, retain the ability to disagree, and later measure where people, rules, and models diverged.

This prototype explores that workflow rather than attempting autonomous care:

- **Traceability over opaque scoring.** Every deterministic escalation has a named rule and recorded input values.
- **Clinician agency over automation.** The reviewer selects the final L0–L3 tier and can agree, modify, or disagree with the system.
- **Evaluation over claims.** The app records reviewer-scoped decisions, timing, agreement, and safety-oriented study metrics.
- **Constrained LLM support over free-form advice.** The optional LLM produces a synthetic-case summary only, with server-verified citations to the daily record.

## Product walkthrough

### 1. Review Workspace

Select reviewer R1 or R2, then choose a case from the reviewer-specific randomized queue. The workspace provides:

- a compact patient snapshot and baseline;
- weight and oxygenation trend;
- a readable day-by-day follow-up trail;
- deterministic rule evidence for each escalation;
- an independent reviewer decision form; and
- an optional evidence-cited LLM research summary.

The API begins timing when a reviewer opens a case. The browser cannot provide its own review duration as the source of truth; the server records elapsed time when a decision is saved.

### 2. Settings

The Settings page demonstrates practical workspace preferences:

- English and Traditional Chinese interface language;
- light and dark display modes;
- a browser-local organisation logo upload;
- screen-sharing identifiers mask;
- confirmation before saving;
- local auto-lock; and
- optional LLM connection status.

Appearance and privacy preferences, including the uploaded logo, remain in the current browser local storage. They are not stored in PostgreSQL.

### 3. Safety Lab

Safety Lab is the evaluation view. It compares:

- deterministic-rule results across the 18 synthetic cases;
- LLM results after a model benchmark has run; and
- submitted reviewer decisions.

It reports evaluation coverage, exact tier matches, L3 emergency sensitivity, L0–L1 specificity, critical misses, evidence-backed LLM cases, and a reviewer disagreement trail. The model benchmark is intentionally held in API memory and is cleared after an API restart, preventing it from being mistaken for a durable clinical record.

## Demo path

1. In **Review Workspace**, open a pending case and inspect the timeline, trend, and rule evidence.
2. Select a final tier and document agreement or disagreement. The decision is saved to PostgreSQL under the selected reviewer code.
3. Open **Safety Lab** to show how the study makes disagreements and safety-oriented metrics visible.
4. If an LLM is configured, return to a case and select **Generate research summary**. Click an evidence chip to jump to its cited daily record.
5. Run the Safety Lab benchmark and show model coverage, citation coverage, L3 sensitivity, and critical-miss count.
6. Close by reiterating the boundary: synthetic research evaluation, not patient care.

The demo sequence above is self-contained so a reviewer can run it directly from the public repository.

## Architecture

~~~mermaid
flowchart LR
    Judge["Reviewer / judge browser"] --> Web["React + Vite<br/>Review Workspace"]
    Web -->|REST + reviewer header| API["Go API<br/>study workflow"]
    API --> Rules["Deterministic L0–L3 rules<br/>evidence-producing"]
    API --> DB[("PostgreSQL<br/>review decisions")]
    API --> Dataset[("18 synthetic HF cases<br/>JSON dataset")]
    API -. optional, server-side only .-> LLM["OpenAI-compatible<br/>LLM endpoint"]
    LLM --> Verify["Strict schema + citation<br/>verification"]
    Verify --> API
    API --> Lab["Safety Lab<br/>rules × LLM × reviewer"]
    Lab --> Web
~~~

| Layer | Responsibility |
| --- | --- |
| React / Vite frontend | Review interface, data visualisation, browser-local preferences, bilingual UI |
| Go API | Case access, reviewer queues, deterministic rules, timing, exports, LLM guardrails, Safety Lab |
| PostgreSQL | Reviewer-scoped decisions and server-measured duration |
| Synthetic JSON dataset | 18 fictional follow-up cases; no PHI |
| Optional LLM endpoint | Research-only structured second reader; never called directly from the browser |

## Deterministic triage engine

The bundled rules-v1.0 engine evaluates every follow-up day and returns L0–L3 assessments plus the named rules that fired. It is deliberately deterministic and inspectable.

| Tier | Research interpretation | Example signals implemented |
| --- | --- | --- |
| L0 | No escalation signal | No trigger reached |
| L1 | Lower-acuity follow-up signal | Borderline oxygenation below baseline; diet or fluid indiscretion |
| L2 | Prompt review signal | Rapid or weekly weight increase, worsening exertional dyspnoea, new PND, rising oedema, mild low oxygenation, tachycardia, elevated SBP, selected adherence/congestion combinations |
| L3 | Emergency-pattern signal | Chest pain, dyspnoea at rest, SpO₂ below 90%, frothy sputum, syncope, high-risk palpitations, symptomatic hypotension, or new confusion |

The cohort contains 18 synthetic cases with designed peak tiers: 3 L0, 3 L1, 6 L2, and 6 L3. Designed answers are study references for prototype evaluation; they are not clinical ground truth.

## Optional LLM research assistant

The application can call an OpenAI-compatible chat/completions endpoint as a constrained **second reader**. It is disabled by default.

Create a local .env file in the repository root. It is ignored by Git:

~~~dotenv
# Required to enable the optional research assistant
LLM_API_KEY=replace_with_your_key
LLM_MODEL=gpt-5.6

# Optional. The server defaults to https://api.openai.com/v1 when omitted.
LLM_BASE_URL=https://api.openai.com/v1
~~~

Rebuild the API:

~~~powershell
docker compose up -d --build
~~~

Confirm the configuration:

~~~powershell
Invoke-RestMethod http://localhost:8080/api/llm/status
~~~

When enabled, the server sends only the selected synthetic case, its check-ins, and deterministic rule assessments. The browser never receives the key, and generated summaries are not persisted to the study database.

### Evidence and safety controls

The LLM integration is designed to make unsupported output harder to present as evidence:

1. The request asks for strict JSON Schema output: risk tier, uncertainty, rule alignment, short rationale, signals, evidence citations, questions, and safety note.
2. Citations are limited to known follow-up fields and actual days in the selected synthetic case.
3. The API resolves each accepted citation against the canonical record and replaces the model value with the canonical value.
4. Invalid, duplicate, or unknown citations are discarded. If no verifiable evidence remains, the response is rejected.
5. The prompt and UI label the output research-only; the LLM cannot save a clinician decision or prescribe an action.

The LLM may be unavailable, incorrect, incomplete, or inconsistent. It never substitutes for independent professional judgement.

## Study workflow and stored data

1. Each seeded reviewer, R1 and R2, receives the same 18 cases in a stable reviewer-specific random order.
2. Opening a case starts server-side timing for that reviewer and case.
3. A reviewer submits a tier from L0 to L3, an alignment of agree, modify, or disagree, and a required note for modifications or disagreements.
4. PostgreSQL stores one current decision per reviewer code and case ID pair.
5. Study coordinators can inspect the summary and export CSV or JSON.

The analytics summary includes reviewer-versus-rule agreement, unweighted and linear-weighted Cohen’s kappa when enough decisions exist, confusion matrix, median review duration, L3 sensitivity, and low-risk specificity.

The workflow is intentionally bounded to synthetic research use: assign reviewer codes outside the application, keep reviewers independent, export only after collection, and never enter real patient information.

## API reference

All endpoints return JSON unless stated otherwise. Reviewer-facing endpoints default to R1 when the X-Reviewer-Code header is omitted.

| Method | Endpoint | Purpose |
| --- | --- | --- |
| GET | /healthz | Service health check |
| GET | /api/cases | Queue for the current reviewer |
| GET | /api/reviewers/{reviewerCode}/queue | Queue for a named seeded reviewer |
| GET | /api/cases/{caseID} | Blinded case detail, rule assessments, and that reviewer’s saved decision |
| POST | /api/cases/{caseID}/open | Start server-side review timing |
| POST | /api/cases/{caseID}/decision | Save a reviewer decision |
| GET | /api/analytics/summary | Agreement, timing, and safety-oriented aggregate metrics |
| GET | /api/analytics/export?format=csv | Download research export as CSV |
| GET | /api/analytics/export?format=json | Retrieve research export as JSON |
| GET | /api/llm/status | Whether optional server-side LLM support is configured |
| POST | /api/cases/{caseID}/llm-assessment | Generate an evidence-cited synthetic-case summary |
| GET | /api/safety-lab | Current Safety Lab report |
| POST | /api/safety-lab/benchmark | Start optional 18-case LLM benchmark |

Example request for reviewer R2:

~~~powershell
Invoke-RestMethod -Headers @{ 'X-Reviewer-Code' = 'R2' } http://localhost:8080/api/cases
~~~

## Testing and verification

### Automated checks

With Go 1.26+ and Node.js installed locally:

~~~powershell
Set-Location backend
go test ./...

Set-Location ..\frontend
npm ci
npm run build
~~~

The Go test suite covers deterministic rules, assignments, review storage and timing, analytics and exports, LLM citation verification, API handlers, and a simulated 18-case Safety Lab benchmark. The frontend build runs TypeScript checking and production bundling.

### Docker smoke test

After starting Docker Compose:

~~~powershell
docker compose ps
(Invoke-RestMethod http://localhost:8080/healthz).status
Invoke-RestMethod http://localhost:8080/api/safety-lab
~~~

Expected baseline:

- health check returns ok;
- Safety Lab reports 18 dataset cases;
- deterministic rules have evaluated all 18 cases;
- model benchmark is idle until an LLM is configured and the benchmark is started.

## Repository map

~~~text
backend/
  cmd/api/                 HTTP API and composition root
  internal/agent/          Deterministic L0–L3 engine
  internal/analytics/      Agreement, kappa, safety metrics, exports
  internal/assignments/    Stable reviewer-specific queues
  internal/llm/            Strict-schema, evidence-verified second reader
  internal/reviews/        Timing and PostgreSQL decision repository
  internal/safety/         Cohort Safety Lab aggregation
  db/migrations/           PostgreSQL schema
frontend/
  src/main.tsx             Bilingual review, settings, and lab interface
  src/styles.css           Responsive visual system
data/
  synthetic_hf_cases.json  18-case synthetic study dataset
Docs/
  READER_STUDY_PROTOCOL.md Proposed study execution procedure
HACKATHON_DEMO.md          Three-minute demo guide
docker-compose.yml         One-command local environment
~~~

## Important boundaries and limitations

- **Synthetic-only:** Do not add real patient information to the dataset, UI, database, logs, prompts, or exports.
- **Not for care:** The tiers, rules, LLM output, and metrics are research artifacts; none recommends treatment or establishes patient safety.
- **No identity or access system:** Reviewer codes are study labels, not authentication. Deployments would require real identity, access control, audit design, and institutional governance.
- **Development database credentials:** The Compose PostgreSQL username and password are intentionally local-development values. They are not production security controls.
- **Local browser preferences:** Masking, auto-lock, theme, and logo settings are usability safeguards only; they do not replace endpoint security.
- **Model variability:** LLM results are optional, non-deterministic across providers or model versions, and stored only in Safety Lab memory for the active API process.
- **Evaluation scope:** This prototype demonstrates a transparent evaluation workflow. It does not establish clinical validity, effectiveness, regulatory compliance, or suitability for patient care.

## Troubleshooting

| Symptom | Check |
| --- | --- |
| localhost:5173 is unavailable | Run docker compose ps, then inspect docker compose logs -f web. The first dependency install can take time. |
| API health check fails | Inspect docker compose logs -f api postgres; the API waits for PostgreSQL’s health check. |
| A port is already in use | Stop the application using that local port, or change the port mapping in docker-compose.yml. |
| LLM button is disabled | Verify .env has both LLM_API_KEY and LLM_MODEL, rebuild, then request /api/llm/status. |
| Safety Lab benchmark fails | Confirm the LLM endpoint and credentials, check API logs, then rerun the benchmark. |
| You need a clean study state | Run docker compose down -v; this deletes local Docker volumes and all saved review decisions. |

## Project intent

HF Readmit Agent is built around a simple proposition: in high-stakes evaluation workflows, useful AI should be inspectable, challengeable, and measurable. The project turns that proposition into a runnable experience—evidence appears beside the signal, a human remains the decision maker, and the Safety Lab exposes disagreement rather than hiding it.