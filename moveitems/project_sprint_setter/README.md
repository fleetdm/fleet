# sprintsetter

Small terminal app to find GitHub Project (v2) items that are missing a Sprint (iteration field),
print them, and (optionally) set them to the *current* iteration.

## Requirements
- Go (1.22+ should work; go.mod uses 1.23.4 like your existing tools)
- `GITHUB_TOKEN` with access to the org + project (classic PAT or fine-grained token with Projects access)

## Build
```bash
go build -o sprintsetter .
```

## Run
```bash
export GITHUB_TOKEN=...   # required
./sprintsetter -org fleetdm -project 71
```

### Common options
- Use a different iteration field name:
```bash
./sprintsetter -project 71 -field Sprint
```

- Dry run:
```bash
./sprintsetter -project 71 -dry-run
```

- Skip prompt (careful):
```bash
./sprintsetter -project 71 -yes
```

- Cap updates (first N items only):
```bash
./sprintsetter -project 71 -limit 25
```

## Notes
- "Current iteration" is chosen by date: startDate <= today < startDate+duration.
  If no iteration spans today, it falls back to the most recent iteration that started in the past.
