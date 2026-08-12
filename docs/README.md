# geopilot documentation

Reference material for talking to a WaterFurnace **AWL (Aurora WebLink)** controller
locally and understanding what its registers mean — recovered by reverse-engineering
the controller's own local interface so an independently-written program (geopilot)
can interoperate with a device the manufacturer's cloud has abandoned.

## Contents

| Doc | What's in it |
|---|---|
| [`local-access.md`](local-access.md) | How to get the controller serving `request.cgi` on your LAN again — the reversible config route (Soft-AP) vs. the firmware patch (true LAN access, required for geopilot). |
| [`protocol/request-cgi.md`](protocol/request-cgi.md) | The `request.cgi` Modbus passthrough: request shape, auth, key diagnostic registers, and the heat-of-rejection/extraction math. |
| [`protocol/registers.md`](protocol/registers.md) | The recovered Aurora register map (125 labelled registers across both UIs). |
| [`protocol/dealer-registers.md`](protocol/dealer-registers.md) | Dealer / IntelliZone-2 register catalog, grouped by page. |
| [`protocol/faults.md`](protocol/faults.md) | Aurora E-codes, VS-drive alarms, and IZ2 zone faults, plus how fault history is stored. |
| [`diagnosis.md`](diagnosis.md) | A worked example: diagnosing one unit's chronic freeze fault and desuperheater behavior from live reads. |
| [`billing-bayou.md`](billing-bayou.md) | What Bayou Energy is, why geopilot uses it for cost/rate, and how to set it up. |

## Scope and honesty

These docs describe an **interface** and record findings from a specific unit. They
do not reproduce the controller's firmware or any proprietary code. Register labels
marked *inferred* were reached through a single resolved local in the minified UI —
good evidence, but glance at a live value before relying on them; *direct* labels are
certain. Values and fault counts shown are illustrative of the reference unit.

**Before you patch or deploy, read [`../SECURITY.md`](../SECURITY.md)** — local access
leaves an effectively unauthenticated read/write control API on your network.

See [`../NOTICE`](../NOTICE) for the affiliation/trademark disclaimer.
