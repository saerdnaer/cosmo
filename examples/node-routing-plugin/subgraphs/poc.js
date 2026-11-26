import { readFileSync } from 'node:fs';
import gql from 'graphql-tag';
import { GraphQLError } from 'graphql';
import { serve } from './serve.js';

const stations = {
  'poc:tv:28106': { name: 'Das Erste', dvbId: 28106 },
  'poc:radio:28402': { name: 'Bayern 3', dvbId: 28402 },
};
const pocID = /^poc:(tv|radio):([0-9]+)$/;
const typeOf = (id) => (id.startsWith('poc:tv:') ? 'PublicationServiceTV' : 'PublicationServiceRadio');
const find = ({ id }) => (stations[id] ? { id, ...stations[id] } : null);

await serve({
  name: 'poc',
  port: 4003,
  typeDefs: gql(readFileSync(new URL('./poc.graphql', import.meta.url), 'utf8')),
  resolvers: {
    Query: {
      nodePoc: (_, { id }) => {
        if (!pocID.test(id)) {
          throw new GraphQLError(`invalid ID ${id}: want poc:<tv|radio>:<dvbId>`);
        }
        return find({ id });
      },
    },
    Node: { __resolveType: ({ id }) => typeOf(id) },
    PublicationServiceTV: { __resolveReference: find },
    PublicationServiceRadio: { __resolveReference: find },
  },
});
