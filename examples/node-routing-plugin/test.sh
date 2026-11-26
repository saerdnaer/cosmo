#!/usr/bin/env bash
# End-to-end check: Query.node is owned by the "node" router plugin, which only
# maps the global ID to a typename. The router must then fetch the remaining
# fields from the subgraph that owns the concrete type.
#
# Usage: ./test.sh            (builds router, plugin and config first)
#        SKIP_BUILD=1 ./test.sh
set -euo pipefail
cd "$(dirname "$0")"

WGC=(npx -y wgc@0.135.0)
ROUTER_BIN=${ROUTER_BIN:-./.bin/router}
LOG_DIR=./.logs
mkdir -p "$LOG_DIR" ./.bin

if [[ -z "${SKIP_BUILD:-}" ]]; then
  (cd ../../router && go build -o "$OLDPWD/.bin/router" ./cmd/router)
  # wgc is run from a neutral directory: the repo root package.json pins a
  # Node version via devEngines, which npx enforces.
  here=$PWD
  (cd "${TMPDIR:-/tmp}" && "${WGC[@]}" router plugin build --yes \
    --go-module-path github.com/wundergraph/cosmo/examples/node-routing-plugin/plugins/node \
    "$here/plugins/node")
  (cd "${TMPDIR:-/tmp}" && "${WGC[@]}" router compose -i "$here/graph.yaml" -o "$here/config.json")
  (cd subgraphs && npm install --no-audit --no-fund --silent)
fi

pids=()
cleanup() { kill "${pids[@]}" 2>/dev/null || true; }
trap cleanup EXIT

(cd subgraphs && exec node cms.js) >"$LOG_DIR/cms.log" 2>&1 & pids+=($!)
(cd subgraphs && exec node comments.js) >"$LOG_DIR/comments.log" 2>&1 & pids+=($!)
"$ROUTER_BIN" -config router.yaml >"$LOG_DIR/router.log" 2>&1 & pids+=($!)

for _ in $(seq 1 60); do
  curl -sf -o /dev/null http://localhost:3002/health/ready && break
  sleep 0.5
done

query() {
  curl -s http://localhost:3002/graphql -H 'content-type: application/json' \
    -d "$(jq -nc --arg q "$1" --arg id "$2" '{query: $q, variables: {id: $id}}')"
}

failed=0
check() { # name, query, id, expected JSON (compared with jq -S)
  local got
  got=$(query "$2" "$3")
  if [[ $(jq -S . <<<"$got") == $(jq -S . <<<"$4") ]]; then
    echo "PASS $1"
  else
    echo "FAIL $1"; echo "  want: $4"; echo "  got:  $got"; failed=1
  fi
}

Q='query($id: ID!) { node(id: $id) { __typename id ... on Article { title } ... on Comment { text article { title } } } }'

check "article routed to cms" "$Q" "kmp:cms:article:1" \
  '{"data":{"node":{"__typename":"Article","id":"kmp:cms:article:1","title":"Hello Federation"}}}'

check "comment routed to comments (and nested article to cms)" "$Q" "kmp:comments:comment:7" \
  '{"data":{"node":{"__typename":"Comment","id":"kmp:comments:comment:7","text":"First!","article":{"title":"Hello Federation"}}}}'

check "unknown domain resolves to null" "$Q" "kmp:epg:broadcast:1" \
  '{"data":{"node":null}}'

got=$(query "$Q" "kmp:cms")
if jq -e '.data.node == null and (.errors | length > 0)' <<<"$got" >/dev/null; then
  echo "PASS malformed id returns an error"
else
  echo "FAIL malformed id returns an error"; echo "  got: $got"; failed=1
fi

echo "--- subgraph calls"
cat "$LOG_DIR/cms.log" "$LOG_DIR/comments.log" | grep -v -e "ready at" -e "inline tracing"
exit $failed
