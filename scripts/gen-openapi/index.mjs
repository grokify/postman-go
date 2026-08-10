// Reconstructs an OpenAPI 3.0.3 specification for the Postman API from the
// official TypeScript SDK sources (github.com/postmanlabs/postman-api-sdk-ts).
//
// Postman does not publish a downloadable OpenAPI document, so this generator
// parses the TS SDK — the authoritative machine-readable source — using the
// TypeScript compiler AST (via ts-morph) and emits ../../openapi/openapi.yaml.
//
// See README.md for the approach and known approximations.

import { Project, SyntaxKind, Node } from 'ts-morph';
import yaml from 'js-yaml';
import path from 'node:path';
import fs from 'node:fs';
import url from 'node:url';

const __dirname = path.dirname(url.fileURLToPath(import.meta.url));
const REPO_ROOT = path.resolve(__dirname, '..', '..');
const TS_SDK =
  process.env.POSTMAN_TS_SDK ||
  path.resolve(REPO_ROOT, '..', '..', 'postmanlabs', 'postman-api-sdk-ts');
const SERVICES_DIR = path.join(TS_SDK, 'src', 'services');
const OUT = path.join(REPO_ROOT, 'openapi', 'openapi.yaml');

const log = (...a) => console.error('[gen]', ...a);
const warnings = [];
const warn = (m) => {
  warnings.push(m);
};

if (!fs.existsSync(SERVICES_DIR)) {
  console.error(`TS SDK services dir not found: ${SERVICES_DIR}`);
  console.error('Set POSTMAN_TS_SDK to the postman-api-sdk-ts checkout.');
  process.exit(1);
}

// ---------------------------------------------------------------------------
// Load project
// ---------------------------------------------------------------------------
const project = new Project({
  compilerOptions: { allowJs: false, skipLibCheck: true },
  skipAddingFilesFromTsConfig: true,
  skipFileDependencyResolution: true,
});
project.addSourceFilesAtPaths(path.join(SERVICES_DIR, '**/*.ts'));
const sourceFiles = project.getSourceFiles();
log(`loaded ${sourceFiles.length} TS source files`);

// ---------------------------------------------------------------------------
// Component registry
//
// Every emitted component has a globally-unique name. Because liblab reuses
// positional names (e.g. SuccessfulResponseData2) across services, we key
// internally by (filePath, declName) and assign a unique final name, suffixing
// on collision. $ref resolution always goes (identifier -> source file ->
// final name), so refs stay correct after suffixing.
// ---------------------------------------------------------------------------
const components = {}; // finalName -> schema object
const usedNames = new Set();

// resolution maps
const baseConstToType = new Map(); // key `${file}::${constName}` -> { typeName, file }
const declToFinal = new Map(); // key `${file}::${declName}` -> finalName (for types, base consts, variants, enums)
const enumByName = new Map(); // simpleEnumName -> [finalNames] (best effort by simple name)

function uniqueName(preferred) {
  let name = preferred.replace(/[^A-Za-z0-9_]/g, '_');
  if (!/^[A-Za-z_]/.test(name)) name = '_' + name;
  if (!usedNames.has(name)) {
    usedNames.add(name);
    return name;
  }
  let i = 2;
  while (usedNames.has(`${name}_${i}`)) i++;
  const finalName = `${name}_${i}`;
  usedNames.add(finalName);
  return finalName;
}

const fileKey = (file, name) => `${file}::${name}`;

// ---------------------------------------------------------------------------
// Pass 1: index enums and base types across all files
// ---------------------------------------------------------------------------
const enumDecls = []; // { file, name, values }
const typeDecls = []; // { file, typeName, baseConst }

for (const sf of sourceFiles) {
  const file = sf.getFilePath();

  for (const en of sf.getEnums()) {
    const values = en
      .getMembers()
      .map((m) => {
        const v = m.getValue();
        return v === undefined ? m.getName() : v;
      })
      .filter((v) => v !== undefined && v !== '');
    enumDecls.push({ file, name: en.getName(), values });
  }

  // export type X = z.infer<typeof constName>
  for (const ta of sf.getTypeAliases()) {
    const typeNode = ta.getTypeNode();
    if (!typeNode) continue;
    const txt = typeNode.getText();
    const m = txt.match(/^z\s*\.\s*infer\s*<\s*typeof\s+([A-Za-z0-9_]+)\s*>$/s);
    if (m) {
      typeDecls.push({ file, typeName: ta.getName(), baseConst: m[1] });
    }
  }
}

// Assign final names: enums first, then types.
for (const e of enumDecls) {
  const finalName = uniqueName(e.name);
  components[finalName] = {
    type: 'string',
    enum: e.values,
  };
  declToFinal.set(fileKey(e.file, e.name), finalName);
  const arr = enumByName.get(e.name) || [];
  arr.push({ finalName, file: e.file });
  enumByName.set(e.name, arr);
}

for (const t of typeDecls) {
  const finalName = uniqueName(t.typeName);
  // map type name, base const, and its Request/Response variants to this component
  declToFinal.set(fileKey(t.file, t.typeName), finalName);
  declToFinal.set(fileKey(t.file, t.baseConst), finalName);
  declToFinal.set(fileKey(t.file, t.baseConst + 'Request'), finalName);
  declToFinal.set(fileKey(t.file, t.baseConst + 'Response'), finalName);
  baseConstToType.set(fileKey(t.file, t.baseConst), { typeName: t.typeName, finalName, file: t.file });
}

log(`indexed ${enumDecls.length} enums, ${typeDecls.length} base types`);

// ---------------------------------------------------------------------------
// Import resolution: given a source file and an identifier, find the file the
// identifier is declared in (following the export chain one hop).
// ---------------------------------------------------------------------------
function resolveIdentifierFile(sf, ident) {
  // local declaration?
  const localVar = sf.getVariableDeclaration(ident);
  if (localVar) return sf.getFilePath();
  const localEnum = sf.getEnum(ident);
  if (localEnum) return sf.getFilePath();
  const localType = sf.getTypeAlias(ident);
  if (localType) return sf.getFilePath();

  for (const imp of sf.getImportDeclarations()) {
    const named = imp.getNamedImports().map((n) => n.getName());
    const def = imp.getDefaultImport()?.getText();
    if (named.includes(ident) || def === ident) {
      const target = imp.getModuleSpecifierSourceFile();
      if (target) return target.getFilePath();
      return null;
    }
  }
  return null;
}

// Resolve an identifier used as a schema/type to a component final name.
function refFor(sf, ident) {
  const targetFile = resolveIdentifierFile(sf, ident) || sf.getFilePath();
  // try direct (file, ident)
  let finalName = declToFinal.get(fileKey(targetFile, ident));
  if (finalName) return finalName;
  // try stripping Request/Response variant suffix
  for (const suf of ['Response', 'Request']) {
    if (ident.endsWith(suf)) {
      finalName = declToFinal.get(fileKey(targetFile, ident.slice(0, -suf.length)));
      if (finalName) return finalName;
    }
  }
  // fall back: search any file for this decl name
  for (const [k, v] of declToFinal) {
    if (k.endsWith(`::${ident}`)) return v;
  }
  return null;
}

// ---------------------------------------------------------------------------
// Zod expression -> JSON Schema
// ---------------------------------------------------------------------------
// Given a ts-morph Expression node representing a zod schema, produce a JSON
// Schema fragment and whether it is optional / nullable.
function parseZod(sf, expr, depth = 0) {
  if (depth > 40) return { schema: {} };

  // unwrap parenthesized
  if (Node.isParenthesizedExpression(expr)) {
    return parseZod(sf, expr.getExpression(), depth + 1);
  }

  // Identifier referring to another schema const -> $ref
  if (Node.isIdentifier(expr)) {
    const name = expr.getText();
    if (name === 'z') return { schema: {} };
    const ref = refFor(sf, name);
    if (ref) return { schema: { $ref: `#/components/schemas/${ref}` } };
    return { schema: {} };
  }

  if (!Node.isCallExpression(expr)) {
    // arrow function body etc.
    return { schema: {} };
  }

  // Walk a call chain: base is z.<kind>(...) then .optional()/.nullable()/...
  const callee = expr.getExpression();
  const method = Node.isPropertyAccessExpression(callee) ? callee.getName() : null;

  const modifiers = { optional: false, nullable: false, default: undefined };
  const chainMods = (m) => {
    if (m === 'optional') modifiers.optional = true;
    else if (m === 'nullable' || m === 'nullish') modifiers.nullable = true;
    if (m === 'nullish') modifiers.optional = true;
  };

  // Handle chained modifiers by recursing into the object side.
  if (method && ['optional', 'nullable', 'nullish', 'default', 'describe', 'catch', 'readonly'].includes(method)) {
    const inner = callee.getExpression();
    const res = parseZod(sf, inner, depth + 1);
    chainMods(method);
    res.optional = res.optional || modifiers.optional;
    res.nullable = res.nullable || modifiers.nullable;
    return res;
  }

  // Base z.<kind>(...)
  if (method && Node.isPropertyAccessExpression(callee)) {
    const obj = callee.getExpression();
    // z.lazy(() => EXPR)
    if (method === 'lazy') {
      const arg = expr.getArguments()[0];
      if (arg && (Node.isArrowFunction(arg) || Node.isFunctionExpression(arg))) {
        const body = arg.getBody();
        let ret = body;
        if (Node.isBlock(body)) {
          const rs = body.getStatements().find((s) => Node.isReturnStatement(s));
          if (rs) ret = rs.getExpression();
        }
        if (ret) return parseZod(sf, ret, depth + 1);
      }
      return { schema: {} };
    }
    // obj may itself be `z` or a nested call (chained). If obj is a call we
    // already handled modifiers above; here obj should be `z`.
    const kind = method;
    switch (kind) {
      case 'string':
        return { schema: { type: 'string' } };
      case 'number':
      case 'bigint':
        return { schema: { type: 'integer' } };
      case 'boolean':
        return { schema: { type: 'boolean' } };
      case 'date':
        return { schema: { type: 'string', format: 'date-time' } };
      case 'any':
      case 'unknown':
      case 'never':
      case 'void':
      case 'undefined':
      case 'null':
        return { schema: {} };
      case 'literal': {
        const a = expr.getArguments()[0];
        if (a && Node.isStringLiteral(a)) return { schema: { type: 'string', enum: [a.getLiteralValue()] } };
        if (a && (Node.isNumericLiteral(a))) return { schema: { type: 'integer' } };
        return { schema: {} };
      }
      case 'enum': {
        const a = expr.getArguments()[0];
        if (a && Node.isArrayLiteralExpression(a)) {
          const vals = a.getElements().filter(Node.isStringLiteral).map((e) => e.getLiteralValue());
          return { schema: { type: 'string', enum: vals } };
        }
        return { schema: { type: 'string' } };
      }
      case 'nativeEnum': {
        const a = expr.getArguments()[0];
        if (a && Node.isIdentifier(a)) {
          const ref = refFor(sf, a.getText());
          if (ref) return { schema: { $ref: `#/components/schemas/${ref}` } };
        }
        return { schema: { type: 'string' } };
      }
      case 'array': {
        const a = expr.getArguments()[0];
        const items = a ? parseZod(sf, a, depth + 1).schema : {};
        return { schema: { type: 'array', items } };
      }
      case 'object': {
        return { schema: parseZodObject(sf, expr, depth) };
      }
      case 'record': {
        const args = expr.getArguments();
        const valArg = args.length === 2 ? args[1] : args[0];
        const ap = valArg ? parseZod(sf, valArg, depth + 1).schema : {};
        return { schema: { type: 'object', additionalProperties: ap && Object.keys(ap).length ? ap : true } };
      }
      case 'union':
      case 'discriminatedUnion': {
        // Represent unions as free-form to keep ogen happy; unions here are
        // dominated by [z.any(), ...] shapes. (Documented approximation.)
        return { schema: {} };
      }
      case 'intersection': {
        const args = expr.getArguments();
        const parts = args.map((a) => parseZod(sf, a, depth + 1).schema).filter((s) => s && Object.keys(s).length);
        if (parts.length) return { schema: { allOf: parts } };
        return { schema: {} };
      }
      case 'instanceof':
      case 'function':
      case 'promise':
        return { schema: {} };
      case 'tuple': {
        return { schema: { type: 'array', items: {} } };
      }
      default:
        // Unknown z.<kind>; if obj is not `z`, maybe it's a ref chain.
        if (Node.isIdentifier(obj) && obj.getText() !== 'z') {
          const ref = refFor(sf, obj.getText());
          if (ref) return { schema: { $ref: `#/components/schemas/${ref}` } };
        }
        return { schema: {} };
    }
  }

  return { schema: {} };
}

function parseZodObject(sf, callExpr, depth) {
  const arg = callExpr.getArguments()[0];
  const schema = { type: 'object' };
  const properties = {};
  const required = [];
  if (arg && Node.isObjectLiteralExpression(arg)) {
    for (const prop of arg.getProperties()) {
      if (!Node.isPropertyAssignment(prop)) continue;
      const nameNode = prop.getNameNode();
      let key;
      if (Node.isStringLiteral(nameNode)) key = nameNode.getLiteralValue();
      else key = prop.getName();
      const valExpr = prop.getInitializer();
      if (!valExpr) continue;
      const res = parseZod(sf, valExpr, depth + 1);
      let fieldSchema = res.schema || {};
      if (res.nullable) {
        if (fieldSchema.$ref) {
          fieldSchema = { nullable: true, allOf: [fieldSchema] };
        } else {
          fieldSchema = { ...fieldSchema, nullable: true };
        }
      }
      properties[key] = fieldSchema;
      if (!res.optional) required.push(key);
    }
  }
  if (Object.keys(properties).length) schema.properties = properties;
  if (required.length) schema.required = required;

  // Best-effort enum enhancement via positional JSDoc @property.
  applyJsDocEnums(sf, callExpr, schema);
  return schema;
}

// Look at the JSDoc @property annotations attached to the enclosing
// `export const`/`export type` and, if the count matches the field count and a
// property names a known enum, upgrade that field to a $ref.
function applyJsDocEnums(sf, callExpr, schema) {
  if (!schema.properties) return;
  try {
    // find enclosing variable statement
    let stmt = callExpr;
    while (stmt && !Node.isVariableStatement(stmt)) stmt = stmt.getParent();
    if (!stmt) return;
    // JSDoc may be on the type alias that follows; search whole file text near.
    const constDecl = stmt.getDeclarations?.()[0];
    if (!constDecl) return;
    const constName = constDecl.getName();
    // the export type <T> = z.infer<typeof constName> carries the @property docs
    const ta = sf.getTypeAliases().find((t) => {
      const tn = t.getTypeNode();
      return tn && tn.getText().includes(`typeof ${constName}`);
    });
    if (!ta) return;
    const docs = ta.getJsDocs();
    if (!docs.length) return;
    const full = docs.map((d) => d.getInnerText()).join('\n');
    const propTypes = [];
    const re = /@property\s*\{([^}]+)\}/g;
    let m;
    while ((m = re.exec(full)) !== null) propTypes.push(m[1].trim());
    const keys = Object.keys(schema.properties);
    if (propTypes.length !== keys.length) return;
    keys.forEach((k, i) => {
      let t = propTypes[i];
      const isArray = /\[\]$/.test(t);
      t = t.replace(/\[\]$/, '').trim();
      if (!enumByName.has(t)) return;
      const cand = enumByName.get(t);
      const ref = `#/components/schemas/${cand[0].finalName}`;
      const cur = schema.properties[k];
      const nullable = cur && cur.nullable;
      const refSchema = nullable ? { nullable: true, allOf: [{ $ref: ref }] } : { $ref: ref };
      if (isArray) {
        schema.properties[k] = { type: 'array', items: { $ref: ref } };
      } else {
        schema.properties[k] = refSchema;
      }
    });
  } catch {
    /* best effort */
  }
}

// ---------------------------------------------------------------------------
// Pass 2: build component schemas from each base const's initializer
// ---------------------------------------------------------------------------
for (const t of typeDecls) {
  const finalName = declToFinal.get(fileKey(t.file, t.typeName));
  if (!finalName || components[finalName]) continue;
  const sf = project.getSourceFile(t.file);
  const decl = sf.getVariableDeclaration(t.baseConst);
  if (!decl) {
    components[finalName] = {};
    continue;
  }
  const init = decl.getInitializer();
  if (!init) {
    components[finalName] = {};
    continue;
  }
  try {
    const res = parseZod(sf, init, 0);
    let s = res.schema || {};
    if (res.nullable) s = { ...s, nullable: true };
    components[finalName] = s;
  } catch (e) {
    warn(`model ${t.typeName} (${path.basename(t.file)}): ${e.message}`);
    components[finalName] = {};
  }
}

// ---------------------------------------------------------------------------
// Error classes -> error components
// ---------------------------------------------------------------------------
const errorClassToRef = new Map();
function refForErrorClass(sf, className) {
  if (errorClassToRef.has(className)) return errorClassToRef.get(className);
  const targetFile = resolveIdentifierFile(sf, className);
  let ref = null;
  if (targetFile) {
    const tf = project.getSourceFile(targetFile);
    // prefer a `<x>Response` const, else any base const in the file
    const consts = tf.getVariableDeclarations().map((d) => d.getName());
    const respConst = consts.find((c) => /Response$/.test(c)) || consts[0];
    if (respConst) {
      // find a type in file to map, else synthesize
      const key = fileKey(targetFile, respConst);
      let finalName = declToFinal.get(key);
      if (!finalName) {
        finalName = uniqueName(className);
        const decl = tf.getVariableDeclaration(respConst);
        try {
          components[finalName] = decl ? parseZod(tf, decl.getInitializer(), 0).schema : { type: 'object' };
        } catch {
          components[finalName] = { type: 'object' };
        }
      }
      ref = finalName;
    }
  }
  if (!ref) {
    // synthesize a permissive component
    ref = usedNames.has(className) ? uniqueName(className) : (usedNames.add(className), className);
    components[ref] = components[ref] || { type: 'object' };
  }
  errorClassToRef.set(className, ref);
  return ref;
}

// ---------------------------------------------------------------------------
// Pass 3: operations
// ---------------------------------------------------------------------------
const paths = {};
let opCount = 0;
const skippedOps = [];

// Resolve a TS type reference (from a param/return annotation) to a component.
function refForTypeText(sf, typeText) {
  let t = typeText.trim();
  // unwrap Promise<>, HttpResponse<>, ArrayBuffer, etc.
  const unwrap = (s) => {
    const mm = s.match(/^([A-Za-z0-9_.]+)\s*<\s*(.+)\s*>$/s);
    return mm ? { outer: mm[1], inner: mm[2].trim() } : null;
  };
  let guard = 0;
  while (guard++ < 5) {
    const u = unwrap(t);
    if (!u) break;
    if (['Promise', 'HttpResponse', 'Array', 'ReadonlyArray'].includes(u.outer)) {
      if (u.outer === 'Array' || u.outer === 'ReadonlyArray') {
        const inner = refForTypeText(sf, u.inner);
        return inner ? { type: 'array', items: inner } : { type: 'array', items: {} };
      }
      t = u.inner;
      continue;
    }
    break;
  }
  if (/\[\]$/.test(t)) {
    const inner = refForTypeText(sf, t.replace(/\[\]$/, ''));
    return inner ? { type: 'array', items: inner } : { type: 'array', items: {} };
  }
  if (['void', 'any', 'unknown', 'never', 'undefined', 'null'].includes(t)) return null;
  if (t === 'string') return { type: 'string' };
  if (t === 'number') return { type: 'integer' };
  if (t === 'boolean') return { type: 'boolean' };
  if (t === 'ArrayBuffer' || t === 'Blob' || t === 'Uint8Array') return { type: 'string', format: 'binary' };
  // identifier type -> component
  const ref = refFor(sf, t);
  if (ref) return { $ref: `#/components/schemas/${ref}` };
  return null;
}

function findChainCalls(chainRoot) {
  // returns array of { name, callExpr } from base to outer
  const calls = [];
  let node = chainRoot;
  while (node && Node.isCallExpression(node)) {
    const callee = node.getExpression();
    if (Node.isPropertyAccessExpression(callee)) {
      calls.push({ name: callee.getName(), call: node });
      node = callee.getExpression();
    } else {
      break;
    }
  }
  return calls.reverse();
}

for (const sf of sourceFiles) {
  const file = sf.getFilePath();
  if (!/-service\.ts$/.test(path.basename(file))) continue;
  const cls = sf.getClasses()[0];
  if (!cls) continue;

  // parse request-params interfaces in sibling request-params.ts
  const paramsFile = project.getSourceFile(path.join(path.dirname(file), 'request-params.ts'));
  const paramIfaces = new Map(); // ifaceName -> [{name, required, type}]
  if (paramsFile) {
    for (const iface of paramsFile.getInterfaces()) {
      const fields = iface.getProperties().map((p) => ({
        name: p.getName(),
        required: !p.hasQuestionToken(),
        type: p.getTypeNode()?.getText() || 'string',
      }));
      paramIfaces.set(iface.getName(), fields);
    }
  }

  for (const method of cls.getMethods()) {
    const opId = method.getName();
    if (/^set[A-Z].*Config$/.test(opId)) continue;

    // find the RequestBuilder chain (a call expression ending in .build())
    let buildCall = null;
    method.forEachDescendant((d) => {
      if (buildCall) return;
      if (Node.isCallExpression(d)) {
        const c = d.getExpression();
        if (Node.isPropertyAccessExpression(c) && c.getName() === 'build') {
          // ensure the chain contains new RequestBuilder()
          if (d.getText().includes('RequestBuilder')) buildCall = d;
        }
      }
    });
    if (!buildCall) continue;

    const chainInner = buildCall.getExpression().getExpression(); // the expr before .build()
    const calls = findChainCalls(chainInner);

    let httpMethod = null;
    let rawPath = null;
    const queryKeys = [];
    for (const { name, call } of calls) {
      const args = call.getArguments();
      if (name === 'setMethod' && args[0] && Node.isStringLiteral(args[0])) httpMethod = args[0].getLiteralValue().toLowerCase();
      else if (name === 'setPath' && args[0] && Node.isStringLiteral(args[0])) rawPath = args[0].getLiteralValue();
      else if (name === 'addQueryParam') {
        const o = args[0];
        if (o && Node.isObjectLiteralExpression(o)) {
          const keyProp = o.getProperty('key');
          if (keyProp && Node.isPropertyAssignment(keyProp)) {
            const kv = keyProp.getInitializer();
            if (kv && Node.isStringLiteral(kv)) queryKeys.push(kv.getLiteralValue());
          }
        }
      }
    }
    if (!httpMethod || !rawPath) {
      skippedOps.push(`${opId}: missing method/path`);
      continue;
    }

    // collect responses/errors
    const responses = {};
    for (const { name, call } of calls) {
      const o = call.getArguments()[0];
      if (!o || !Node.isObjectLiteralExpression(o)) continue;
      if (name === 'addResponse' || name === 'addError') {
        const statusProp = o.getProperty('status');
        let status = name === 'addResponse' ? '200' : 'default';
        if (statusProp && Node.isPropertyAssignment(statusProp)) {
          const sv = statusProp.getInitializer();
          if (sv && Node.isNumericLiteral(sv)) status = String(sv.getLiteralValue());
        }
        if (name === 'addError') {
          const errProp = o.getProperty('error');
          let ref = null;
          if (errProp && Node.isPropertyAssignment(errProp)) {
            const ev = errProp.getInitializer();
            if (ev && Node.isIdentifier(ev)) ref = refForErrorClass(sf, ev.getText());
          }
          responses[status] = {
            description: httpStatusText(status),
            content: { 'application/json': { schema: ref ? { $ref: `#/components/schemas/${ref}` } : { type: 'object' } } },
          };
        }
      }
    }

    // 200 response schema from return type
    let okSchema = null;
    const rt = method.getReturnTypeNode();
    if (rt) okSchema = refForTypeText(sf, rt.getText());
    if (okSchema) {
      responses['200'] = {
        description: 'Successful Response',
        content: { 'application/json': { schema: okSchema } },
      };
    } else if (!responses['200']) {
      responses['200'] = { description: 'Successful Response' };
    }

    // path params
    const pathParamNames = [...rawPath.matchAll(/\{([^}]+)\}/g)].map((m) => m[1]);

    // params interface for query typing
    const ifaceName = opId.charAt(0).toUpperCase() + opId.slice(1) + 'Params';
    const ifaceFields = paramIfaces.get(ifaceName) || [];
    const ifaceByName = new Map(ifaceFields.map((f) => [f.name, f]));

    const parameters = [];
    // ogen cannot distinguish parameters whose names collide after
    // normalization (e.g. `orderBy` vs `order_by` both -> Go OrderBy). Track
    // seen normalized names and drop later collisions. (Documented approximation.)
    const seenNorm = new Set();
    const norm = (s) => s.toLowerCase().replace(/[_-]/g, '');
    for (const pn of pathParamNames) {
      seenNorm.add(norm(pn));
      parameters.push({ name: pn, in: 'path', required: true, schema: { type: 'string' } });
    }
    for (const qk of queryKeys) {
      if (seenNorm.has(norm(qk))) {
        warn(`${opId}: dropped query param '${qk}' (name collides with existing param under ogen normalization)`);
        continue;
      }
      seenNorm.add(norm(qk));
      const f = ifaceByName.get(qk);
      let schema = { type: 'string' };
      if (f) schema = tsTypeToParamSchema(sf, f.type);
      parameters.push({ name: qk, in: 'query', required: f ? f.required : false, schema });
    }

    // request body: method param named 'body'
    const bodyParam = method.getParameters().find((p) => p.getName() === 'body');
    let requestBody;
    if (bodyParam) {
      const bt = bodyParam.getTypeNode()?.getText();
      let bs = bt ? refForTypeText(sf, bt) : null;
      if (!bs) bs = { type: 'object' };
      requestBody = {
        required: !bodyParam.hasQuestionToken(),
        content: { 'application/json': { schema: bs } },
      };
    }

    const op = {
      operationId: opId,
      tags: [serviceTag(file)],
      summary: firstSentence(method),
    };
    if (parameters.length) op.parameters = parameters;
    if (requestBody) op.requestBody = requestBody;
    op.responses = responses;

    if (!paths[rawPath]) paths[rawPath] = {};
    paths[rawPath][httpMethod] = op;
    opCount++;
  }
}

function tsTypeToParamSchema(sf, typeText) {
  const t = typeText.trim();
  if (t === 'number') return { type: 'integer' };
  if (t === 'boolean') return { type: 'boolean' };
  if (t === 'string') return { type: 'string' };
  if (/\[\]$/.test(t)) return { type: 'array', items: { type: 'string' } };
  // enum type?
  const ref = refFor(sf, t);
  if (ref && components[ref] && components[ref].enum) return { $ref: `#/components/schemas/${ref}` };
  return { type: 'string' };
}

function serviceTag(file) {
  const dir = path.basename(path.dirname(file));
  return dir
    .replace(/_$/, '')
    .split('-')
    .map((s) => s.charAt(0).toUpperCase() + s.slice(1))
    .join('');
}

function firstSentence(method) {
  const docs = method.getJsDocs();
  if (!docs.length) return method.getName();
  const txt = docs[0].getDescription().trim().split(/\n/)[0].trim();
  return txt || method.getName();
}

function httpStatusText(code) {
  const map = {
    '400': 'Bad Request',
    '401': 'Unauthorized',
    '403': 'Forbidden',
    '404': 'Not Found',
    '409': 'Conflict',
    '422': 'Unprocessable Entity',
    '429': 'Too Many Requests',
    '500': 'Internal Server Error',
    default: 'Error',
  };
  return map[code] || `Status ${code}`;
}

log(`built ${opCount} operations across ${Object.keys(paths).length} paths`);
if (skippedOps.length) log(`skipped ${skippedOps.length} operations: ${skippedOps.join('; ')}`);

// ---------------------------------------------------------------------------
// Prune $refs that point to missing components (safety) and assemble document
// ---------------------------------------------------------------------------
function fixRefs(node) {
  if (Array.isArray(node)) {
    node.forEach(fixRefs);
  } else if (node && typeof node === 'object') {
    if (typeof node.$ref === 'string') {
      const name = node.$ref.split('/').pop();
      if (!components[name]) {
        delete node.$ref;
        node.type = node.type || 'object';
      }
    }
    for (const k of Object.keys(node)) fixRefs(node[k]);
  }
}
fixRefs(paths);
fixRefs(components);

const doc = {
  openapi: '3.0.3',
  info: {
    title: 'Postman API',
    description:
      'Go client surface for the Postman API. Reconstructed from the official ' +
      'Postman TypeScript SDK (postmanlabs/postman-api-sdk-ts) by scripts/gen-openapi. ' +
      'Postman does not publish a downloadable OpenAPI document.',
    version: '0.1.0',
    license: { name: 'MIT', url: 'https://opensource.org/licenses/MIT' },
  },
  servers: [
    { url: 'https://api.postman.com', description: 'Default (US) region' },
    { url: 'https://api.eu.postman.com', description: 'EU region' },
  ],
  security: [{ apiKey: [] }],
  paths: sortKeys(paths),
  components: {
    securitySchemes: {
      apiKey: {
        type: 'apiKey',
        in: 'header',
        name: 'X-API-Key',
        description: 'A Postman API key.',
      },
    },
    schemas: sortKeys(components),
  },
};

function sortKeys(obj) {
  const out = {};
  for (const k of Object.keys(obj).sort()) out[k] = obj[k];
  return out;
}

fs.mkdirSync(path.dirname(OUT), { recursive: true });
fs.writeFileSync(
  OUT,
  '# Generated by scripts/gen-openapi from the Postman TypeScript SDK. DO NOT EDIT BY HAND.\n' +
    '# Regenerate: node scripts/gen-openapi/index.mjs\n' +
    yaml.dump(doc, { noRefs: true, lineWidth: 120, sortKeys: false }),
);

log(`wrote ${OUT}`);
log(`components: ${Object.keys(components).length}, operations: ${opCount}`);
if (warnings.length) log(`warnings: ${warnings.length} (model parse fallbacks)`);
