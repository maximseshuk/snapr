import { defineConfig } from 'oxlint'

export default defineConfig({
  plugins: ['typescript', 'react', 'oxc', 'unicorn'],
  env: {
    browser: true,
    es2022: true,
  },
  categories: {
    correctness: 'error',
    suspicious: 'warn',
  },
  rules: {
    'no-unused-vars': 'off',
    'typescript/no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
    'react-hooks/rules-of-hooks': 'error',
    'react-hooks/exhaustive-deps': 'warn',
    'react/react-in-jsx-scope': 'off',
    'sort-imports': 'off',
    'no-shadow': 'off',
  },
  ignorePatterns: ['dist', 'node_modules', 'src/routeTree.gen.ts'],
})
