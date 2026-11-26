import { readFileSync } from 'node:fs';
import gql from 'graphql-tag';
import { serve } from './serve.js';

const documents = {
  'kmp:cms:mnsoYQFYSOK2EJdaYa7VGQ': { __typename: 'Article', title: 'Hello Federation' },
  'kmp:cms:jLPeT7neT0OBx1d-5ZlLDA': { __typename: 'PublicationServiceCmsTV', name: 'hr', title: 'hr-fernsehen' },
};
const find = (id, typename) => {
  const doc = documents[id];
  return doc && (!typename || doc.__typename === typename) ? { id, ...doc } : null;
};

await serve({
  name: 'cms-dgs',
  port: 4001,
  typeDefs: gql(readFileSync(new URL('./cms.graphql', import.meta.url), 'utf8')),
  resolvers: {
    Query: { nodeCms: (_, { id }) => find(id) },
    Node: { __resolveType: (doc) => doc.__typename },
    PublicationService: { __resolveType: (doc) => doc.__typename },
    Article: { __resolveReference: ({ id }) => find(id, 'Article') },
    PublicationServiceCmsTV: { __resolveReference: ({ id }) => find(id, 'PublicationServiceCmsTV') },
  },
});
