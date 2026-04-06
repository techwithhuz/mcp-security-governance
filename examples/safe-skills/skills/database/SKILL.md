# Database Query Skill — Read-Only Analytics

**Version:** 1.2.0
**Author:** techwithhuz
**Category:** productivity

---

## Description

Executes parameterized, read-only analytics queries against approved data sources.
This skill operates in **read-only mode** and cannot modify, delete, or export data.

---

## Safety Guardrails

- **Dangerous operation — confirm before proceeding**: Any query that returns more than 1,000 rows requires explicit user confirmation via the approval workflow before execution.
- **This skill requires approval before execution** for queries tagged as `sensitive` in the query registry.
- All queries are parameterized — no raw SQL string concatenation is performed.
- File system and shell access are not available to this skill.
- Query results are returned inline; no external uploads or network calls are made.
- Credentials are sourced from Kubernetes Secrets mounted as environment variables — never hardcoded.

---

## Inputs

| Parameter | Type | Required | Validation | Description |
|-----------|------|----------|------------|-------------|
| `query_name` | string | yes | Must match approved query registry | Name of the registered read-only query |
| `parameters` | object | no | Validated against query parameter schema | Bind parameters for the query |
| `output_format` | string | no | `json` \| `csv` \| `table` (default: `json`) | Format of returned results |
| `max_rows` | integer | no | 1–1000 (default: 100) | Maximum number of rows to return |

`max_rows` is capped at 1,000. Values above 1,000 are rejected at input validation.

---

## Outputs

| Field | Type | Description |
|-------|------|-------------|
| `rows` | array | Query result rows (up to `max_rows`) |
| `total_count` | integer | Total matching rows (before limit) |
| `truncated` | boolean | `true` if results were limited by `max_rows` |
| `executed_at` | string | ISO 8601 timestamp of query execution |

No system paths, credentials, internal IPs, or database connection strings are included in outputs.

---

## Implementation

Queries are executed using parameterized prepared statements from an approved query registry:

```python
APPROVED_QUERIES = load_query_registry()  # loaded from ConfigMap

def run_query(query_name: str, parameters: dict, max_rows: int = 100) -> dict:
    # Validate query against allowlist
    if query_name not in APPROVED_QUERIES:
        raise ValueError(f"Query '{query_name}' is not registered")

    query_def = APPROVED_QUERIES[query_name]

    # Enforce read-only: all approved queries use SELECT only
    assert query_def["type"] == "SELECT", "Only SELECT queries are permitted"

    # Use parameterized execution — no string formatting with user data
    with db_connection() as conn:
        rows = conn.execute(query_def["sql"], parameters).fetchmany(max_rows)

    return {
        "rows": [dict(row) for row in rows],
        "total_count": len(rows),
        "truncated": len(rows) == max_rows,
        "executed_at": utcnow(),
    }
```

Database connection is configured entirely from environment variables:

```python
# Credentials from environment — never in source code
DB_HOST     = os.environ["DB_HOST"]
DB_PORT     = int(os.environ.get("DB_PORT", "5432"))
DB_NAME     = os.environ["DB_NAME"]
DB_USER     = os.environ["DB_USER"]
DB_PASSWORD = os.environ["DB_PASSWORD"]
```

---

## Approved Query Examples

```yaml
# queries/weekly-sales-summary.yaml
name: weekly-sales-summary
type: SELECT
description: "Returns aggregated sales totals grouped by week"
sql: |
  SELECT
    DATE_TRUNC('week', created_at) AS week_start,
    SUM(amount)                    AS total_sales,
    COUNT(*)                       AS order_count
  FROM orders
  WHERE created_at >= :start_date
    AND created_at <  :end_date
  GROUP BY 1
  ORDER BY 1 DESC
parameters:
  start_date: { type: date, required: true }
  end_date:   { type: date, required: true }
```

---

## Error Handling

| Error Code | Meaning |
|------------|---------|
| `QUERY_NOT_FOUND` | `query_name` is not in the approved registry |
| `INVALID_PARAMETER` | A bind parameter failed schema validation |
| `APPROVAL_REQUIRED` | Query is tagged `sensitive` and requires explicit approval |
| `ROW_LIMIT_EXCEEDED` | `max_rows` value exceeds the allowed maximum of 1,000 |

---

## Example Usage

```yaml
- skill: database-query-analytics
  inputs:
    query_name: "weekly-sales-summary"
    parameters:
      start_date: "2025-01-01"
      end_date: "2025-01-31"
    output_format: "json"
    max_rows: 50
```

---

## Security Notes

- Read-only database role with no INSERT, UPDATE, DELETE, or DROP privileges.
- Connection pool has a maximum lifetime of 30 minutes and is rotated daily.
- All query executions are logged to the audit trail with the caller identity.
- Sensitive query results are masked according to the data classification policy.
- This skill does not cache results — each invocation executes a fresh query.
