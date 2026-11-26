import { readFileSync } from 'node:fs';
import gql from 'graphql-tag';
import { serve } from './serve.js';

const comments = {
  'kmp:comments:d6ZfvSNnRgCkwa69rhYUZQ': {
    id: 'kmp:comments:d6ZfvSNnRgCkwa69rhYUZQ',
    text: 'First!',
    article: { id: 'kmp:cms:mnsoYQFYSOK2EJdaYa7VGQ' },
  },
};

await serve({
  name: 'comments',
  port: 4002,
  typeDefs: gql(readFileSync(new URL('./comments.graphql', import.meta.url), 'utf8')),
  resolvers: {
    Comment: { __resolveReference: ({ id }) => comments[id] ?? null },
  },
});
