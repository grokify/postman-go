> Postman Secret Scanner - is there a Go SDK or OpenAPI Spec / Postman Collection for the Postman Secret Scanner API

Yes—partially:

* **Go SDK:** I found **no official Go SDK** for the Postman API/Secret Scanner. The repository named `postman-sdk-go` is deprecated and unrelated—it captured HTTP traffic for Live Collections.
* **Official SDK:** Postman now provides an official **TypeScript SDK**, including `SecretScannerService`: [postmanlabs/postman-api-sdk-ts](https://github.com/postmanlabs/postman-api-sdk-ts).
* **Postman Collection:** Yes. The official **Postman API collection** contains a `Secret Scanner` folder with the relevant requests: [Secret Scanner requests](https://www.postman.com/postman/postman-public-workspace/folder/4m4etpx/detected-secrets). You can fork/copy the collection into your workspace.
* **OpenAPI:** Postman has a public API definition underlying its generated TypeScript SDK and documentation, but I could not find a stable, directly downloadable raw OpenAPI URL specifically for Secret Scanner. The definition covers the broader Postman API rather than being a standalone Secret Scanner specification.

The documented Secret Scanner operations are:

* `GET /secret-types`
* `POST /detected-secrets-queries`
* `GET /detected-secrets/{secretId}/locations`
* `PUT /detected-secrets/{secretId}` to update resolution status

Authentication uses `X-API-Key` against `https://api.getpostman.com`. Access currently requires a Postman Enterprise plan with the Advanced Security Administration add-on, and permissions depend on the caller’s admin/workspace role. [Official Secret Scanner documentation](https://learning.postman.com/docs/administration/managing-your-team/secret-scanner/overview)

For Go, the practical choices are either a small hand-written `net/http` client for these four endpoints or generating a client after exporting the Postman API definition/collection to OpenAPI.
