import typescript from '@typescript-eslint/eslint-plugin';
import typescriptParser from '@typescript-eslint/parser';
import importPlugin from 'eslint-plugin-import';
import promise from 'eslint-plugin-promise';
import react from 'eslint-plugin-react';

export default [
  {
    plugins: {
      react,
      '@typescript-eslint': typescript,
      import: importPlugin,
      promise
    }
  },
  {
    files: ['**/*.{tsx,jsx}'],
    languageOptions: {
      parser: typescriptParser
    },
    settings: {
      react: {
        version: 'detect'
      }
    },
    rules: {
      ...react.configs['recommended'].rules,
      'react/jsx-sort-props': [
        'error',
        {
          reservedFirst: true
        }
      ]
    }
  },
  {
    files: ['**/*.ts'],
    languageOptions: {
      parser: typescriptParser
    },
    rules: {
      ...typescript.configs['recommended'].rules,
      ...typescript.configs['eslint-recommended'].rules
    }
  },
  {
    files: ['**/*.{js,ts,tsx}'],
    settings: {
      'import/resolver': {
        typescript: true,
        node: true
      }
    },
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-unused-vars': [
        'error',
        {
          argsIgnorePattern: '^_$'
        }
      ],
      'sort-imports': [
        'error',
        {
          ignoreDeclarationSort: true
        }
      ],
      ...importPlugin.configs['recommended'].rules,
      'import/order': [
        'error',
        {
          pathGroups: [
            {
              pattern: '{react,react-dom/**,react-router-dom}',
              group: 'builtin',
              position: 'before'
            },
            {
              pattern: '@mui/**',
              group: 'external',
              position: 'after'
            }
          ],
          pathGroupsExcludedImportTypes: ['builtin'],
          alphabetize: {
            order: 'asc'
          },
          'newlines-between': 'always'
        }
      ],
      ...promise.configs['recommended'].rules
    }
  }
];
