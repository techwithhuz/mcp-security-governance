# Automation Skill — Task Scheduler

**Version:** 1.2.0
**Author:** techwithhuz
**Category:** productivity

---

## Description

Schedules and executes predefined automation tasks based on user-provided trigger conditions.
All tasks are executed from a fixed, reviewed list — no dynamic code execution is permitted.

---

## Safety Guardrails

- **This skill requires human approval before executing any scheduled task.**
- Tasks are restricted to a pre-approved allowlist maintained by the platform team.
- No shell commands, system calls, or file system access are performed.
- All inputs are validated against an explicit schema before processing.
- This skill does not make external network calls; results are returned inline.

---

## Inputs

| Parameter | Type | Required | Validation | Description |
|-----------|------|----------|------------|-------------|
| `task_name` | string | yes | Must match `/^[a-zA-Z0-9_-]{3,64}$/` | Name of the approved task to schedule |
| `trigger_time` | string | yes | ISO 8601 format | When to run the task |
| `parameters` | object | no | Keys must be in task's allowed parameter list | Optional task parameters |
| `notify_email` | string | no | Valid RFC 5321 email | Email address for completion notification |

All inputs are validated at the boundary. Requests with unrecognized fields are rejected with HTTP 400.

---

## Outputs

| Field | Type | Description |
|-------|------|-------------|
| `task_id` | string | Unique ID assigned to the scheduled task |
| `scheduled_at` | string | ISO 8601 timestamp when task was registered |
| `status` | string | `scheduled` \| `rejected` |

No sensitive data (credentials, internal paths, system info) is included in outputs.

---

## Implementation

Tasks are dispatched by looking up `task_name` in the approved task registry:

```python
APPROVED_TASKS = load_approved_task_registry()  # loaded from ConfigMap, not hardcode

def schedule_task(task_name: str, trigger_time: str, parameters: dict) -> dict:
    # Validate task against allowlist
    if task_name not in APPROVED_TASKS:
        raise ValueError(f"Task '{task_name}' is not in the approved task registry")

    # Validate parameters against task schema
    task_schema = APPROVED_TASKS[task_name]["parameter_schema"]
    validate_against_schema(parameters, task_schema)

    # Schedule via internal task bus (no external HTTP calls)
    task_id = task_bus.enqueue(task_name, trigger_time, parameters)
    return {"task_id": task_id, "scheduled_at": utcnow(), "status": "scheduled"}
```

Credentials are loaded from environment variables at startup, never hardcoded:

```python
# Correct: read from environment
TASK_BUS_TOKEN = os.environ["TASK_BUS_TOKEN"]
DATABASE_URL = os.environ["DATABASE_URL"]
```

---

## Error Handling

| Error Code | Meaning |
|------------|---------|
| `TASK_NOT_FOUND` | `task_name` is not in the approved registry |
| `INVALID_PARAMETER` | A parameter value failed schema validation |
| `INVALID_TRIGGER_TIME` | `trigger_time` is not a valid ISO 8601 timestamp |
| `PERMISSION_DENIED` | Caller does not have permission to schedule this task |

---

## Example Usage

```yaml
- skill: task-scheduler
  inputs:
    task_name: "generate-weekly-report"
    trigger_time: "2025-02-01T09:00:00Z"
    parameters:
      report_type: "summary"
      output_format: "pdf"
    notify_email: "ops-team@example.com"
```

---

## Security Notes

- Tasks run in a sandboxed environment with no access to the host filesystem.
- The approved task registry is managed by the platform team and audited quarterly.
- All scheduling actions are logged to the audit trail.
- Rate limiting: 100 schedule requests per user per hour.
