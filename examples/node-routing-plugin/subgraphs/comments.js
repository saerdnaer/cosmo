import { readFileSync } from 'node:fs';
import gql from 'graphql-tag';
import { serve } from './serve.js';

const comments = {
  'kmp:comments:comment:7': {
    id: 'kmp:comments:comment:7',
    text: 'First!',
    article: { id: 'kmp:cms:article:1' },
  },
};

await serve({
  name: 'comments',
  port: 4002,
  typeDefs: gql(readFileSync(new URL('./comments.graphql', import.meta.url), 'utf8')),
  resolvers: {
    Query: { comment: (_, { id }) => comments[id] ?? null },
    Comment: { __resolveReference: ({ id }) => comments[id] ?? null },
  },
});
