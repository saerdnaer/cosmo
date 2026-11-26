# Routing root fields by ID with a router plugin

Spike for [wundergraph/cosmo#1595](https://github.com/wundergraph/cosmo/issues/1595):
root fields such as Relay's `node(id: ID!): Node` or `publicationService(id: ID!): PublicationService`
return objects that live in different subgraphs. The right subgraph follows from the ID.
This example routes them with a router plugin and no change to the router.

## How it works

```
client ── node(id: "kmp:comments:d6ZfvSNnRgCkwa69rhYUZQ") ─▶ router
router ── QueryNode(id) ──────────────────────────────────▶ node plugin → { __typename: Comment, id }
router ── _entities([{__typename: Comment, id}]) ─────────▶ comments    → { text, article { id } }
router ── _entities([{__typename: Article, id}]) ─────────▶ cms         → { title }
```

- The [`node` plugin](plugins/node) owns `Query.node` and `Query.publicationService`.
  It declares the return interfaces and a stub of each implementing type with
  `@key(fields: "id", resolvable: false)`. It never loads data: it works out the
  concrete type of the ID and returns `{ id }` in the matching `oneof` branch.
- [`cms`](subgraphs/cms.graphql), [`comments`](subgraphs/comments.graphql) and
  [`poc`](subgraphs/poc.graphql) are ordinary subgraphs that own their types as
  resolvable entities.
- The query planner sees that the plugin cannot resolve fields like `title` or
  `name` and adds `_entities` fetches to the owning subgraph — standard
  federation, no custom hook.
- Each root field accepts only the types of its own return interface:
  `publicationService` with an article ID returns `null`.

The plugin runs as a subprocess of the router, so it costs no extra deployment
and no extra network hop compared to a dedicated routing subgraph.

## From ID to type

The plugin knows which subgraph owns an ID, not which type it is. Its
[routing table](plugins/node/routing.yaml) maps an ID prefix to a subgraph:

```yaml
routes:
  kmp:cms:
    subgraph: cms-dgs
  kmp:comments:
    subgraph: comments-dgs
  poc:
    subgraph: poc-dgs
    resolver: publicationServicePoc
```

The subgraph names are the real ones of the Kompass graph.

- The prefix is `kmp:<domain>` for Kompass IDs (`kmp:<domain>[:<type>]:<base64url reference>`)
  and the first segment for any other ID (`poc:tv:28106` -> `poc`).
  [`src/kompassid`](plugins/node/src/kompassid) is a Go port of the Java parser that
  defines the Kompass ID format, so malformed Kompass IDs fail before any call.
- `subgraph` names a subgraph of the graph. On start, the router passes every
  subgraph's routing URL, after `overrides`, to the plugin in its `startup_config`
  (see "Router change" below). `url` overrides it, e.g. for a service outside the graph.
  The plugin refuses to start if a route names an unknown subgraph or a non-HTTP
  one (gRPC subgraphs such as `dns:///sport-dgs:9090` have no GraphQL endpoint).
- The plugin then asks the owning subgraph directly (not via the router):
  `query($id: ID!) { <resolver>(id: $id) { __typename } }`. `resolver` defaults to
  `node` + the capitalised last prefix segment (`nodeCms`, `nodeComments`) and can
  be overridden per route (`publicationServicePoc`).
- Each subgraph decides the rest: the type, `null` for unknown IDs, or an error
  for IDs it rejects. The lookup fields are `@inaccessible`, so clients never see them.
- `${VAR}` / `${VAR:-default}` are taken from the environment. The file is read from
  `$NODE_ROUTING_CONFIG`, or from `routing.yaml` in the plugin directory.

| ID | Result |
|---|---|
| prefix with a route | whatever the route's resolver returns |
| valid `kmp:` ID, domain without a route | `null` |
| malformed `kmp:` ID, or unknown non-`kmp` prefix | error |

Each ID costs two calls to the owning subgraph: the type lookup and the
`_entities` fetch.

## Router change

Plugins so far only received tracing and IP anonymization settings. This branch
adds the graph's subgraphs to the plugin `startup_config`:

```json
{"subgraphs": [{"name": "cms-dgs", "routing_url": "http://cms-dgs:8080/graphql"}]}
```

The URLs are the ones the router itself uses, so stage-specific
`overrides.subgraphs.<name>.routing_url` apply to the lookups too. With a router
without this change, set `url` on every route.

## Run

Requires Go, Node.js, `jq` and `curl`.

```bash
./test.sh                # builds router, plugin and config, then runs the checks
SKIP_BUILD=1 ./test.sh   # reuse earlier build output
```

The script prints every call each subgraph received.

## Limits

- The plugin must declare a stub for every type its fields can return, so adding
  such a type to a subgraph means updating the plugin schema as well.
- The type lookup adds one subgraph call per ID. Routes could later declare a fixed
  type (e.g. `kmp:comments` → `Comment`) to skip it.
- Batch variants such as `nodes(ids: [ID!]!)` would work the same way with a list
  response.
