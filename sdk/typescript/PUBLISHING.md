# Publishing `@dingalwa/atb-sdk`

These commands prepare and publish the TypeScript SDK to npm from `sdk/typescript/`.

## Build and verify

```bash
npm run build
npm pack --dry-run
```

## Publish

```bash
npm publish --access public --tag beta
```

The package metadata points back to the main repository at <https://github.com/pcguest/atb>.
