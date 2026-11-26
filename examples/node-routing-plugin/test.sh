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

for sg in cms comments poc; do
  (cd subgraphs && exec node "$sg.js") >"$LOG_DIR/$sg-dgs.log" 2>&1 & pids+=($!)
done
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

# IDs follow the Kompass ID format (kmp:<domain>:<base64url ref>, see the
# gg.kompass:id library) plus the programme service's legacy poc:<tv|radio>:<dvbId>.
ARTICLE=kmp:cms:mnsoYQFYSOK2EJdaYa7VGQ
CMS_TV=kmp:cms:jLPeT7neT0OBx1d-5ZlLDA
COMMENT=kmp:comments:d6ZfvSNnRgCkwa69rhYUZQ
POC_TV=poc:tv:28106
POC_RADIO=poc:radio:28402

NODE='query($id: ID!) { node(id: $id) { __typename id ... on Article { title } ... on Comment { text article { title } } ... on PublicationService { name } } }'
PS='query($id: ID!) { publicationService(id: $id) { __typename id name ... on PublicationServiceCmsTV { title } ... on PublicationServiceTV { dvbId } ... on PublicationServiceRadio { dvbId } } }'

echo "--- node"
check "cms article" "$NODE" "$ARTICLE" \
  "{\"data\":{\"node\":{\"__typename\":\"Article\",\"id\":\"$ARTICLE\",\"title\":\"Hello Federation\"}}}"
check "comment routed to comments, nested article to cms" "$NODE" "$COMMENT" \
  "{\"data\":{\"node\":{\"__typename\":\"Comment\",\"id\":\"$COMMENT\",\"text\":\"First!\",\"article\":{\"title\":\"Hello Federation\"}}}}"
check "poc radio station" "$NODE" "$POC_RADIO" \
  "{\"data\":{\"node\":{\"__typename\":\"PublicationServiceRadio\",\"id\":\"$POC_RADIO\",\"name\":\"Bayern 3\"}}}"
check "unknown cms reference resolves to null" "$NODE" "kmp:cms:AAAAAAAAAAAAAAAAAAAAAQ" '{"data":{"node":null}}'
check "domain without subgraph resolves to null" "$NODE" "kmp:event:YWJj" '{"data":{"node":null}}'

echo "--- publicationService"
check "cms station" "$PS" "$CMS_TV" \
  "{\"data\":{\"publicationService\":{\"__typename\":\"PublicationServiceCmsTV\",\"id\":\"$CMS_TV\",\"name\":\"hr\",\"title\":\"hr-fernsehen\"}}}"
check "poc tv station" "$PS" "$POC_TV" \
  "{\"data\":{\"publicationService\":{\"__typename\":\"PublicationServiceTV\",\"id\":\"$POC_TV\",\"name\":\"Das Erste\",\"dvbId\":28106}}}"
check "poc radio station" "$PS" "$POC_RADIO" \
  "{\"data\":{\"publicationService\":{\"__typename\":\"PublicationServiceRadio\",\"id\":\"$POC_RADIO\",\"name\":\"Bayern 3\",\"dvbId\":28402}}}"
check "article id is not a publication service" "$PS" "$ARTICLE" '{"data":{"publicationService":null}}'

echo "--- errors"
for bad in kmp:cms kmp:nope:YWJj nope:tv:1 poc:tv:abc; do
  got=$(query "$NODE" "$bad")
  if jq -e '.data.node == null and (.errors | length > 0)' <<<"$got" >/dev/null; then
    echo "PASS malformed id $bad returns an error"
  else
    echo "FAIL malformed id $bad returns an error"; echo "  got: $got"; failed=1
  fi
done

for field in nodeCms nodeComments publicationServicePoc; do
  got=$(query "query(\$id: ID!) { $field(id: \$id) { id } }" "$ARTICLE")
  if jq -e --arg f "$field" '.errors[0].message | test($f)' <<<"$got" >/dev/null; then
    echo "PASS $field is hidden from clients"
  else
    echo "FAIL $field is hidden from clients"; echo "  got: $got"; failed=1
  fi
done

echo "--- subgraph calls"
grep -hv "ready at" "$LOG_DIR"/{cms,comments,poc}-dgs.log
exit $failed
