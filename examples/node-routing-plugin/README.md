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
    url: ${CMS_SUBGRAPH_URL:-http://localhost:4001/graphql}
  kmp:comments:
    url: ${COMMENTS_SUBGRAPH_URL:-http://localhost:4002/graphql}
  poc:
    url: ${POC_SUBGRAPH_URL:-http://localhost:4003/graphql}
```

- The prefix is `kmp:<domain>` for Kompass IDs (`kmp:<domain>[:<type>]:<base64url reference>`)
  and the first segment for any other ID (`poc:tv:28106` → `poc`).
  [`src/kompassid`](plugins/node/src/kompassid) is a Go port of the Java parser that
  defines the Kompass ID format, so malformed Kompass IDs fail before any call.
- The plugin then asks the owning subgraph directly (not via the router):
  `query($id: ID!) { node<Domain>(id: $id) { __typename } }`. The field name
  defaults to `node` + the capitalised last prefix segment (`nodeCms`,
  `nodeComments`, `nodePoc`) and can be set per route with `field`.
- Each subgraph decides the rest: the type, `null` for unknown IDs, or an error
  for IDs it rejects. The lookup fields are `@inaccessible`, so clients never see them.
- `${VAR}` / `${VAR:-default}` are taken from the environment. The file is read from
  `$NODE_ROUTING_CONFIG`, or from `routing.yaml` in the plugin directory.

| ID | Result |
|---|---|
| prefix with a route | whatever `node<Domain>` returns |
| valid `kmp:` ID, domain without a route | `null` |
| malformed `kmp:` ID, or unknown non-`kmp` prefix | error |

Each ID costs two calls to the owning subgraph: the type lookup and the
`_entities` fetch.

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
- Subgraph URLs live both in the router config and in the routing table.
- The type lookup adds one subgraph call per ID. Routes could later declare a fixed
  type (e.g. `kmp:comments` → `Comment`) to skip it.
- Batch variants such as `nodes(ids: [ID!]!)` would work the same way with a list
  response.
