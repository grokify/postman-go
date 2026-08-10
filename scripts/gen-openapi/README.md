# gen-openapi

Reconstructs an OpenAPI 3.0.3 specification for the Postman API from the
official Postman TypeScript SDK
([postmanlabs/postman-api-sdk-ts](https://github.com/postmanlabs/postman-api-sdk-ts)).

Postman does not publish a downloadable OpenAPI document (the definition that
generates its official SDKs lives in a private bucket). To keep the grokify
SDK-generation pipeline consistent across providers — source the spec when it
exists, otherwise author our own — this tool parses the TypeScript SDK, which is
the authoritative machine-readable source, and emits `../../openapi/openapi.yaml`.
That spec then drives the ogen-generated Go client under `../../internal/api/`.

## Usage

```bash
cd scripts/gen-openapi
npm install
cd ../..
node scripts/gen-openapi/index.mjs
./generate.sh          # runs ogen over the regenerated spec
```

By default the generator looks for the TypeScript SDK at
`../../postmanlabs/postman-api-sdk-ts` relative to the repo. Override with:

```bash
POSTMAN_TS_SDK=/path/to/postman-api-sdk-ts node scripts/gen-openapi/index.mjs
```

## How it works

Uses [ts-morph](https://ts-morph.com/) (the TypeScript compiler AST) to parse:

- **Operations** — each `*-service.ts` class method's `new RequestBuilder()...build()`
  chain yields the HTTP method (`setMethod`), path (`setPath`), query params
  (`addQueryParam`), request body (the `body` parameter's type), the 200 response
  (the method's `Promise<T>` return type), and error responses (`addError`).
- **Parameters** — required-ness and types come from the sibling
  `request-params.ts` interfaces; path params come from the `{...}` templates.
- **Schemas** — each model's base Zod schema (`export const x = z.lazy(() => z.object({...}))`)
  is translated to a JSON Schema. `export enum` declarations become string enums.
- **Refs** — resolved through the import graph (identifier → source file → that
  file's component), so components stay correct even when names are suffixed to
  avoid collisions.

## Known approximations

The reconstruction is faithful for routing, field names, and structure, but the
TypeScript source is lossy in a few ways. Where the source is ambiguous the
generator makes a documented, ogen-safe choice:

- **Enums on response fields** — the Zod models encode enum-typed fields as plain
  `z.string()`; the enum type name appears only in positional JSDoc `@property`
  annotations. The generator upgrades a field to an enum `$ref` only when the
  `@property` count exactly matches the field count and the annotation names a
  known enum; otherwise the field stays a plain `string`. Enum *parameters* (typed
  in `request-params.ts`) are always precise.
- **`z.number()` → `integer`** — TypeScript `number` is ambiguous; Postman numeric
  fields are overwhelmingly counts/IDs, so integer is the default.
- **`z.union` / `z.discriminatedUnion`** — emitted as a free-form schema (`{}` →
  `any` in Go). The unions in this SDK are dominated by `[z.any(), ...]` shapes,
  so little is lost.
- **`z.any` / `z.unknown` / `z.record`** — free-form object / `additionalProperties`.
- **Colliding parameter names** — ogen cannot distinguish parameters whose names
  are equal after normalization (e.g. `orderBy` vs `order_by`); the later one is
  dropped and a warning is logged.
- **Component name collisions** — liblab reuses positional names (e.g.
  `SuccessfulResponseData2`) across services; duplicates are suffixed (`_2`, ...).

Run the generator to see the current warning list on stderr.
