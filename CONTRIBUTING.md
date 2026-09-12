# Contributing

SWF is developed through small, reviewable changes. Before submitting a change:

```bash
gofmt -w .
go vet ./...
go test -race ./...
```

Public packages should remain small and documented. Put unstable implementation
details in `internal/`. Any public API change needs tests and documentation;
architecture changes need an ADR in `docs/adr/`.

Never commit credentials, production configuration, personal data, or generated
secrets. Please report security issues using `docs/SECURITY.md` instead of a
public issue.
