# Routing `Query.node` with a router plugin

Spike for [wundergraph/cosmo#1595](https://github.com/wundergraph/cosmo/issues/1595):
a Relay-style `node(id: ID!): Node` root field whose objects live in different
subgraphs, routed by the global ID and without changing the router.

## How it works

```
client ── node(id: "kmp:comments:comment:7") ──▶ router
router ── QueryNode(id) ──────────────────────▶ node plugin   → { __typename: Comment, id }
router ── _entities([{__typename: Comment, id}]) ▶ comments   → { text, article { id } }
router ── _entities([{__typename: Article, id}]) ▶ cms        → { title }
```

- The [`node` plugin](plugins/node) owns `Query.node`. It declares `interface Node`
  and a stub of each implementing type with `@key(fields: "id", resolvable: false)`.
  It never loads data: it maps `<prefix>:<domain>:<type>:<ref>` to a typename
  and returns `{ id }` in the matching `oneof` branch.
- [`cms`](subgraphs/cms.graphql) and [`comments`](subgraphs/comments.graphql) are
  ordinary subgraphs. Each owns its `Node` types as resolvable entities.
- The query planner sees that `Article.title` / `Comment.text` are not available in
  the plugin and adds `_entities` fetches to the owning subgraph — standard
  federation, no custom hook.

Because the plugin runs as a subprocess of the router, it costs no extra
deployment and no extra network hop compared to a dedicated "node" subgraph.

## Run

Requires Go, Node.js, `jq` and `curl`.

```bash
./test.sh                # builds router, plugin and config, then runs the checks
SKIP_BUILD=1 ./test.sh   # reuse earlier build output
```

The script prints the `_entities` calls each subgraph received.

## Limits

- The plugin must declare a stub for every type `node` can return, so adding a
  `Node` type to a subgraph means updating the plugin schema too.
- The typename must be derivable from the ID. IDs without a type part would need
  the plugin to ask the owning subgraph first.
- `node` is a single root field: a `nodes(ids: [ID!]!)` variant would work the
  same way with a list response.
