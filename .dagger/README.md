# CI BUILD

```bash
dagger call -m ./.dagger build \
  --src "." \
  export --path=/tmp/go/build/claim-machinery-api/ \
  --progress plain -vv
```
