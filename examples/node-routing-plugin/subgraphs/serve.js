import { ApolloServer } from '@apollo/server';
import { startStandaloneServer } from '@apollo/server/standalone';
import { ApolloServerPluginInlineTraceDisabled } from '@apollo/server/plugin/disabled';
import { buildSubgraphSchema } from '@apollo/subgraph';

// Starts a federation subgraph and logs every operation it receives, so the
// test can show which subgraph the router actually called.
export async function serve({ name, port, typeDefs, resolvers }) {
  const server = new ApolloServer({
    schema: buildSubgraphSchema({ typeDefs, resolvers }),
    plugins: [
      ApolloServerPluginInlineTraceDisabled(),
      {
        async requestDidStart({ request }) {
          console.log(`[${name}] ${request.query.replace(/\s+/g, ' ').trim()}`);
        },
      },
    ],
  });
  const { url } = await startStandaloneServer(server, { listen: { port } });
  console.log(`[${name}] ready at ${url}`);
}
