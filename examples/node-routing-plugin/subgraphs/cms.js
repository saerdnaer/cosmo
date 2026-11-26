import { readFileSync } from 'node:fs';
import gql from 'graphql-tag';
import { serve } from './serve.js';

const articles = {
  'kmp:cms:article:1': { id: 'kmp:cms:article:1', title: 'Hello Federation' },
};

await serve({
  name: 'cms',
  port: 4001,
  typeDefs: gql(readFileSync(new URL('./cms.graphql', import.meta.url), 'utf8')),
  resolvers: {
    Query: { article: (_, { id }) => articles[id] ?? null },
    Article: { __resolveReference: ({ id }) => articles[id] ?? null },
  },
});
