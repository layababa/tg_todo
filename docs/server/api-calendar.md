# Calendar API

## GET /cal/:token/todo.ics

Retrieve the ICS calendar feed for a specific user identified by the token.

### Parameters

| Name  | Type   | In   | Description                            |
| ----- | ------ | ---- | -------------------------------------- |
| token | string | path | The secure calendar token for the user |

### Responses

- `200 OK`: Returns the `.ics` file content with Content-Type `text/calendar`.
- `404 Not Found`: Invalid token or user not found.

## POST /api/user/calendar-token

Generate or rotate the calendar token for the current user.

### Responses

- `200 OK`:

```json
{
  "token": "generated-uuid-token",
  "url": "webcal://api.example.com/cal/generated-uuid-token/todo.ics"
}
```
