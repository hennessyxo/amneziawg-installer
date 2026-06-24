## Summary

<!-- What does this change and why? Link any related issue (e.g. Closes #12). -->

## Type of change

- [ ] Bug fix
- [ ] New feature
- [ ] Refactor / cleanup
- [ ] Docs
- [ ] Other

## How it was tested

<!-- Commands you ran, and anything verified by hand. If it touches the live
     VPN / SSH path, say whether it was tested on a real server. -->

```
go build ./... && go test ./... -race
shellcheck amneziawg-install.sh   # if the installer changed
```

## Checklist

- [ ] CI is green (build, tests, gofmt, shellcheck)
- [ ] User-facing text and docs updated in both languages (RU and EN) where relevant
- [ ] No secrets, keys, or tokens in the diff
- [ ] The change is focused on one thing
