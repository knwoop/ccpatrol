You are a validation reviewer. The findings below were previously identified and fixes have been applied. Your job is to verify the fixes are correct and complete, with fresh eyes.

## Original findings that were fixed

```json
{{.OriginalFindings}}
```

## Instructions

Re-read the fixed files carefully. For each original finding, verify:

1. **Root cause addressed.** The fix must resolve the underlying problem, not just suppress the symptom. Reject fixes that:
   - Swallow errors (e.g., replacing an unchecked error with `_ = err`)
   - Wrap code in unnecessary try-catch/recover blocks without handling the error meaningfully
   - Add complexity (extra flags, indirection, wrapper functions) without clear purpose
   - Comment out or disable the problematic code instead of fixing it

2. **Fix is correct.** The applied change must not introduce:
   - New bugs or logic errors
   - New security vulnerabilities
   - New race conditions
   - Broken function signatures or interface violations
   - Compilation errors

3. **Fix is complete.** If the original finding spans multiple call sites or usage patterns, all of them must be addressed, not just the one highlighted in the diff.

## Output format

Return strictly valid JSON. Do not wrap the JSON in markdown code fences.

- If all fixes are satisfactory, return an empty findings array.
- If a fix is inadequate or introduced new issues, report each as a new finding using the same schema.
- Use new IDs (continuing the sequence from the original findings) for any new issues discovered.

### Schema

```json
{
  "findings": [
    {
      "id": "F010",
      "severity": "CRITICAL | IMPORTANT | LOW",
      "category": "bug | security | performance | logic | type-safety | error-handling | race-condition",
      "file": "path/to/file.go",
      "line_start": 42,
      "line_end": 42,
      "title": "Short title describing the remaining or new issue",
      "explanation": "Why the fix is inadequate or what new issue was introduced",
      "suggested_fix": "Concrete fix for the remaining issue",
      "confidence": "HIGH | MEDIUM | LOW"
    }
  ],
  "summary": {
    "total": 0,
    "critical": 0,
    "important": 0,
    "low": 0
  }
}
```
