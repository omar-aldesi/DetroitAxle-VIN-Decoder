# Build-Number Fork / Range System — Design

Status: **planning** (no code yet). Branch: `fork-system`.
Scope of this document: **server side only.** Frontend is a later, separate document.

---

## 1. The problem

Every fact we store about a car belongs to one of two tiers:

- **Build-key tier (shared).** Fixed by VIN positions 1–8 / 10 / 11 (WMI + VDS + model
  year + plant). One value, identical for every VIN in the group. Examples: year, make,
  model, body type, drive type, country, engine (displacement / cylinders / config),
  fuel type, doors, and usually trim/series. These already live on the `vehicles` row and
  **do not change** with this system.

- **Build-number tier (forked).** Facts that can change along the **serial** (VIN
  positions 12–17, the last 6 digits). These are *not* encoded anywhere in the build key,
  so two cars that share a build key can legitimately differ. Examples: brake code, rotor
  sizes, suspension, steering — plus whatever the team adds per make/model later.

Today the build-number-tier fields live as plain columns on `vehicles`, which is wrong: a
single value is shared across the whole build key even though it varies per unit. This
system gives the build-number tier a correct home: **per-field ranges of serials.**

> The data source is intentionally open — manual agent entry, a DNR member, an API, a bulk
> import of an existing DB. Sources are pluggable; see §7. (GM Parts Giant is out of scope.)

---

## 2. The model in one picture

- For a given **build key**, **each fork field has its own independent list of ranges.**
- A **range** = `[serial_start, serial_end] = value` for one field.
- **VIN view:** resolve each field independently → the range (for that field) whose span
  covers the VIN's serial.
- **Build-key view:** for each field, the full list of ranges and their values, e.g.
  ```
  brake_code:        [1–100]=JP9   [101–end]=JP6
  front_rotor_size:  [1–end]=350mm
  rear_suspension:   (no data yet)
  ```
- A range is only ever asserted from **≥2 verified VIN sightings** (the assumption) **or**
  an **explicit human/manual range entry**. A single VIN sighting is *pending*, not a range.
- **Unverified data changes nothing. Ever.**

That is the entire model. Everything else is detail.

---

## 3. Core concepts & vocabulary

| Term | Meaning |
|------|---------|
| **Build key** | `vin[0:8] + vin[9:11]` (positions 1–8, 10, 11). Existing grouping key. |
| **Build number / serial** | `vin[11:17]` (positions 12–17), parsed as an integer. |
| **Fork field** | A registered build-number-tier field key (e.g. `brake_code`). |
| **Pending point** | A single verified VIN sighting `(serial, value)` not yet part of a range. Held, but never returned as data (rule #1). |
| **Range** | A confirmed `[start, end] = value` span for one field. Born from ≥2 agreeing points or a manual entry. |
| **Origin** | How a range was created: `vin` (inferred from sightings) or `manual` (entered as an explicit span). |
| **Fork / split** | A new range boundary because a verified value differs from an established range. |
| **Reinforce** | A verified sighting matching an existing range's value → evidence++, no new range. |
| **Assumption** | A VIN-origin range's span is inferred from its endpoints; it holds until a verified contradiction breaks it. |
| **Confidence** | A score derived from the evidence (count / density / interpolation vs. extrapolation). |
| **Source** | Free-form provenance string (`"agent"`, `"dnr"`, `"import:acme"`, …). |

---

## 4. The assumption system

The interpolation is the whole point. It is the **reinforce-vs-split** behavior plus the
**two-point rule.**

- One verified sighting `000001 = JP9` → a **pending point**. *Not yet a range* (rule #1).
- A second verified sighting `000100 = JP9` (same value) → the two points agree, so a
  **confirmed range** is born: `[1–100] = JP9`, `observations = 2`. Every unseen serial
  2–99 is now **assumed** JP9 by interpolation.
- A later verified sighting `000101 = JP6` inside/adjacent to that range → a contradiction.
  Once it is confirmed (its own second point, or a manual entry), it **splits**:
  `[1–100] = JP9`, `[101–end] = JP6`.

**Manual ranges skip the two-point rule.** A DNR member can state directly
`1–100 = JP9`, `101–end = JP6`; those become confirmed `manual`-origin ranges immediately
and are authoritative (see precedence, §6.3).

**Confidence** (per field, derived in `Resolve`):

- `observed` — serial within `[min_seen, max_seen]` of a range with `observations ≥ 2`
  (or any `manual` range) → solid.
- `assumed` — within the range but thin evidence (wide gap relative to span).
- `extrapolated` — serial beyond `max_seen` of a `vin` range → weakest.
- `unknown` — no range and no usable points for this field.

---

## 5. Data model (schema)

Three new tables. No existing build-key columns change. The six existing build-number
columns on `vehicles` are migrated (§9) then kept read-only as backup for one release.

### 5.1 `fork_field` — the registry (the router)

Decides which keys are build-number-tier vs. build-key-tier, optionally scoped by
make/model. **Lives in the DB so admins extend it without a code change.**

```
fork_field
  id           bigserial PK
  key          text   NOT NULL          -- "brake_code", "front_rotor_size", …
  scope_make   text   NULL              -- NULL = all makes
  scope_model  text   NULL              -- NULL = all models for the make
  value_type   text   NOT NULL          -- "string" (v1) | "number" | "enum"
  enabled      bool   NOT NULL DEFAULT true
  description  text
  created_at, updated_at
  UNIQUE (key, scope_make, scope_model)
```

**Default seeded keys** (global, `value_type = "string"`):
`brake_code, front_rotor_size, rear_rotor_size, front_suspension, rear_suspension, steering`.
Optional, off by default: `rear_axle_ratio, transfer_case`.

The fork-field set for a vehicle = enabled rows matching `scope_make IS NULL`
∪ `(make, NULL)` ∪ `(make, model)`.

### 5.2 `field_range` — confirmed ranges, **per field**

```
field_range
  id              bigserial PK
  build_key       text   NOT NULL
  field_key       text   NOT NULL
  serial_start    bigint NOT NULL        -- 0 = base (build-key-wide)
  serial_end      bigint NULL            -- NULL = open-ended (last range)
  value           text   NOT NULL
  value_norm      text   NOT NULL        -- normalized, used for equality/uniqueness
  origin          text   NOT NULL        -- "vin" | "manual"
  source          text   NOT NULL        -- provenance string
  observations    int    NOT NULL        -- confirming verified sightings (manual = 1+)
  serial_min_seen bigint NOT NULL
  serial_max_seen bigint NOT NULL
  boundary_exact  bool   NOT NULL DEFAULT false  -- false = boundary is an approximation
  last_actor      bigint NULL
  created_at, updated_at
  UNIQUE (build_key, field_key, serial_start)
  INDEX  (build_key, field_key, serial_start)
```

Invariants per `(build_key, field_key)`: ranges are non-overlapping, ordered by
`serial_start`; effective coverage of a range is `[serial_start, next.serial_start - 1]`
(or `∞` for the last); at most one base range (`serial_start = 0`).

### 5.3 `field_point` — pending sightings (working state, not an audit log)

Holds only verified single sightings that have **not yet** been absorbed into a range.
Self-pruning: a point is deleted once it helps form / reinforce a range.

```
field_point
  id          bigserial PK
  build_key   text   NOT NULL
  field_key   text   NOT NULL
  serial      bigint NOT NULL
  value       text   NOT NULL
  value_norm  text   NOT NULL
  source      text   NOT NULL
  actor       bigint NULL
  created_at
  UNIQUE (build_key, field_key, serial)
  INDEX  (build_key, field_key, value_norm)
```

> This is **not** the full per-VIN audit log we ruled out — it only holds un-absorbed
> pending points, bounded and transient.

### 5.4 Notes & edits (later phases — schema additions only)

- `agent_notes`: add `origin_serial bigint NULL`. A note is anchored to a serial; its
  "applies / ⚠ verify" badge is computed at read time from field boundaries (a note
  applies to a viewed VIN when no fork boundary, in any field, lies between the two
  serials; `NULL` serial or no range data → ⚠). Nothing is hidden — only annotated.
- `vehicle_field_history`: add `origin_serial bigint NULL` and `tier text`
  (`"build_key"` | `"build_number"`).

---

## 6. Services

### 6.1 `RecordPoint` — a single verified VIN sighting

```
RecordPoint(in PointInput) (Result, error)

PointInput { BuildKey, Serial int64, FieldKey, Value, Source string,
             Verified bool, ActorID *uint, Make, Model string }
Result.Outcome ∈ { Ignored, NotForkField, Pending, RangeCreated, Reinforced, Forked }
```

1. **`Verified == false` → `Ignored`.** No table touched (rule #2).
2. **`FieldKey` not registered for make/model → `NotForkField`.** Caller treats as a
   build-key edit. Registry is the router.
3. Normalize the value (§6.4). Find the **confirmed range** covering `Serial`
   (largest `serial_start ≤ serial`):
   - **covers, same `value_norm`** → **Reinforce** (`observations++`, widen `min/max_seen`).
   - **covers, different value** → record a `field_point` as a held-aside **exception**.
     It does **not** split yet. If there are now **≥2 agreeing exception points inside the
     same covering range**, **carve** a new range whose span is `[min, max]` of those
     exception points, *inside* the existing range — a 3-way split:
     `[r.start, inner.start-1] = old`, `[inner.start, inner.end] = new`,
     `[inner.end+1, r.end] = old` (empty pieces omitted), `boundary_exact = false`; delete
     the consumed points → `Forked`. Otherwise → `Pending`.
   - **no covering range** → if there are now **≥2 agreeing pending points** that are all
     uncovered → create a **confirmed `vin` range** spanning their `[min, max]`, delete the
     consumed points → `RangeCreated`. Otherwise store the point and → `Pending` (rule #1).
   - Points that would straddle a different-valued range boundary are left pending (no
     overlapping range is ever created); the reconciliation worker resolves these later.

### 6.2 `RecordRange` — an explicit human/manual span

```
RecordRange(BuildKey, SerialStart, SerialEnd int64, FieldKey, Value, Source string,
            Verified bool, ActorID *uint, Make, Model string) (Result, error)
```

Verified-only; registry-checked. Creates/replaces a **confirmed `manual` range**
`[start, end] = value` directly (no two-point rule). Overlapping `vin` ranges are trimmed
or superseded per precedence (§6.3); `boundary_exact = true`. This is how DNR enters
`1–100 = JP9`, `101–end = JP6`.

### 6.3 `Resolve` — the read door

```
Resolve(buildKey, serial) → map[fieldKey]Resolved
ResolveByBuildKey(buildKey) → map[fieldKey][]RangeView   // the field → [ranges] table (#3)

Resolved { Value, Source, Origin, Confidence, Observations, RangeID,
           SerialMinSeen, SerialMaxSeen }
```

For each fork field, pick the range covering `serial`. **Precedence on overlap:**
`manual` > `vin`; within the same origin, higher `observations`, then most recent.
Pending `field_point`s are **never** returned as values (rule #1) — at most surfaced as an
"N unconfirmed sighting(s)" hint. `ResolveByBuildKey` returns every confirmed range per
field for the build-key view.

### 6.4 Value normalization & events

- **Normalization:** `value_norm = upper(collapse_spaces(trim(value)))`. All equality and
  uniqueness use `value_norm`, so `"JP9"` and `"jp9 "` never spawn a phantom fork. `value`
  keeps the original display form.
- **Event:** every confirming write emits `ForkChanged{build_key, field_key, serial,
  outcome}` for optional subscribers (audit, reindex, notifications). Core writes don't
  depend on subscribers — this is the "Django-signal" hook.

---

## 7. Source-agnostic design & the rules

- **Two doors, every source uses them:** `RecordPoint` (per-VIN sightings) and
  `RecordRange` (explicit spans). A new source = a new `Source` string + a loop of calls.
  No schema work, no special cases.
- **Rule 1 — two points (or a manual range) to assert a range.** A single VIN sighting is
  pending and never shown as data.
- **Rule 2 — unverified data does not exist.** `Verified == false` is a hard no-op.
- **Rule 3 — verified-only mutates structure.** Only verified data creates / reinforces /
  splits / forms ranges.
- **Rule 4 — provenance always recorded.** Every range and point carries `source`/`actor`.

### Example flows

```
agent enters & verifies a rotor size (per-VIN sighting):
  RecordPoint{ BuildKey, Serial:s, FieldKey:"front_rotor_size",
               Value:"350mm", Source:"agent", Verified:true, ActorID:&id }

DNR enters explicit spans:
  RecordRange{ BuildKey, 1,   100, "brake_code", "JP9", "dnr", true, &id }
  RecordRange{ BuildKey, 101, 0/*open*/, "brake_code", "JP6", "dnr", true, &id }

bulk import of an existing DB:
  for row → RecordPoint{…, Source:"import:acme", Verified:true}   // or RecordRange if spans
```

---

## 8. Integration with the existing edit path

`UpdateVehicle` becomes **tier-aware**:

1. For each edited field, ask the registry: fork field for this make/model?
2. **Fork field →** `RecordPoint` (serial from the VIN; `Verified` from the role/verify
   step). Unverified follows rule #2 — proposal only, no structural effect.
3. **Build-key field →** unchanged: update the `vehicles` column + history as today.

`vehicle_field_history` gains `origin_serial` + `tier` to record range-scoped vs. group-wide.

---

## 9. Migration of existing data

Consistent with rule #2:

1. For each existing vehicle, for each of the six build-number columns
   (`brake_code, front_rotor_size, rear_rotor_size, front_spring_type, rear_spring_type,
   steering_type`): if non-empty **and already verified** (latest history entry verified) →
   seed a **manual base range** (`serial_start = 0`, `serial_end = NULL`, origin
   `"manual"`, source `"legacy"`). Otherwise drop it (unverified ⇒ "did not exist").
2. Keep the six columns physically present, **read-only backup**, for one release; reads
   come from `Resolve`.
3. A later release removes the columns once `Resolve` is the confirmed source of truth.

Nothing *verified* is lost, nothing *unverified* enters, reversible.

---

## 10. Conflict handling (no self-heal)

- Splits/forks are append-only; we never merge ranges or rewrite below a boundary.
- VIN-origin boundaries are approximate (`boundary_exact = false`) until tightened by
  denser data or a manual range.
- A range whose `serial_max_seen` falls outside its post-split coverage is a **flag**, not
  an auto-fix.
- Admin action: **"rebuild ranges for a build key / field"** — deterministic offline
  recompute from stored points + manual ranges.
- A future **background worker** scans flags and reconciles. Off the hot path so the site
  stays fast.

---

## 11. Phasing (no timeline)

- **Phase 1 — Core engine. ✅ DONE.**
  - `fork_field`, `field_range`, `field_point` tables + AutoMigrate; seeds the six defaults
    (`SeedDefaultForkFields`).
  - `RecordPoint`, `RecordRange`, `Resolve`, `ResolveByBuildKey`; `NormalizeForkValue`;
    `ForkChanged` emit (no subscribers yet). Engine behind a `ForkStore` interface.
  - Tests (in-memory store): two-point rule, the `001/100/101/352` walkthrough, carve-inside,
    manual ranges + trim/precedence, reinforce, normalization, unverified no-op.
- **Phase 2 — Tier-aware edit dispatch. ✅ DONE (additive, non-breaking).**
  - `vehicle_field_history` gains `origin_serial` + `tier`.
  - `UpdateVehicle`: fork-field edits tagged `build_number`; trusted ones feed the engine
    (`FeedForkField`). Existing column writes/UI behavior unchanged (dual-write transition).
  - `VerifyEntry`: verifying a previously-untrusted fork-field entry feeds the engine.
  - Read endpoint `GET /api/fork/:vin` (build-key field→ranges view + per-VIN resolve).
  - **Deferred:** the §9 backfill of verified legacy column values — will be an explicit
    admin action (Phase 4), so the engine starts clean and fills via edits/verifies.
- **Phase 3 — Notes become range-aware. ✅ DONE.**
  - `agent_notes.origin_serial`; `AddNote` captures the serial from the VIN.
  - `services.ForkBoundaries` + `NoteScope` (applies only when both serials known, range
    data exists, and no boundary lies between them — else ⚠).
  - `GetVehicle` annotates each note's `scope` for the VIN being viewed; all notes still
    returned. (Frontend rendering of the ⚠ badge is part of the later frontend doc.)
- **Phase 4 — Sources & write endpoints. ✅ DONE (core).**
  - `POST /api/fork/:vin/point` — a verified VIN sighting (two-point rule). Trusted-only.
  - `POST /api/fork/:vin/range` — an authoritative manual span (DNR use). Trusted-only.
  - `GET /api/fork/:vin` now also returns `pending` (per-field un-absorbed sightings).
  - **Deferred:** admin legacy-column backfill (needs the verified-only vs. import-all
    policy decided) and admin "rebuild ranges".
- **Phase 5 — Reconciliation worker** (later) — scans split conflicts, offline rebuild.
- **Frontend** — separate document, after the server phases land.

---

## 12. Open decisions (small, settle during Phase 1)

1. **Base sentinel:** `serial_start = 0` (current) vs. nullable + `is_base` flag. Leaning `0`.
2. **Confidence thresholds:** gap/observation cutoffs for `observed` vs `assumed`. Pin in tests.
3. **Non-numeric serials:** treat the build key as **un-forkable** (build-key-wide only). Confirm.
4. **Boundary placement on disagreeing VIN points:** at the later serial,
   `boundary_exact = false`. Confirm.

---

## 13. Non-goals (out of scope for now)

- GM Parts Giant or any specific external data source (sources are pluggable via the two
  Record doors).
- Frontend / UI work.
- Self-healing / automatic range merging.
- Per-VIN row storage. We store ranges (+ transient pending points), never one row per VIN —
  this keeps the DB bounded: range count ≈ build keys × fields × real production breaks.
