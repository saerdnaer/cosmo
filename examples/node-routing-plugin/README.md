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

The IDs follow the Kompass ID format (`kmp:<domain>[:<type>]:<base64url reference>`).
[`src/kompassid`](plugins/node/src/kompassid) is a Go port of the parsing rules of the
Java library that defines the format.

| ID | Type | How |
|---|---|---|
| `kmp:comments:<ref>` | `Comment` | from the domain |
| `poc:tv:<dvbId>` / `poc:radio:<dvbId>` | `PublicationServiceTV` / `PublicationServiceRadio` | from the prefix |
| `kmp:cms:<ref>` | `Article`, `PublicationServiceCmsTV`, … | asked from cms |
| other valid `kmp:` domains | — | `null` |
| malformed | — | error |

CMS IDs carry no type, so the plugin asks the cms subgraph directly via
`nodeCms(id: ID!): Node`. That field is `@inaccessible`, so clients cannot see it.
A CMS ID therefore costs two cms calls: the type lookup and the `_entities` fetch.

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
- IDs without a type need a lookup in the owning subgraph (see CMS above).
- Batch variants such as `nodes(ids: [ID!]!)` would work the same way with a list
  response.
