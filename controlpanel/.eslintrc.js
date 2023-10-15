module.exports = {
    "env": {
        "browser": true,
        "es2021": true
    },
    "extends": [
        "eslint:recommended",
        "plugin:@typescript-eslint/recommended",
        "plugin:react/recommended",
        "plugin:import/recommended",
        "plugin:promise/recommended"
    ],
    "overrides": [
        {
            "env": {
                "node": true
            },
            "files": [
                ".eslintrc.{js,cjs}"
            ],
            "parserOptions": {
                "sourceType": "script"
            }
        }
    ],
    "parser": "@typescript-eslint/parser",
    "parserOptions": {
        "project": "./tsconfig.json",
        "ecmaVersion": "latest",
        "sourceType": "module"
    },
    "plugins": [
        "@typescript-eslint",
        "react",
    ],
    "rules": {
        "@typescript-eslint/no-explicit-any": 0,
        "@typescript-eslint/no-unused-vars": [
            "error",
            {
                "argsIgnorePattern": "^_"
            }
        ],
        "sort-imports": [
            "error",
            {
                "ignoreDeclarationSort": true
            }
        ],
        "import/order": [
            "error",
            {
                "pathGroups": [
                    {
                        "pattern": "{react,react-dom/**,react-router-dom}",
                        "group": "builtin",
                        "position": "before"
                    },
                    {
                        "pattern": "@mui/**",
                        "group": "external",
                        "position": "after"
                    }
                ],
                "pathGroupsExcludedImportTypes": ["builtin"],
                "alphabetize": {
                    "order": "asc"
                },
                "newlines-between": "always"
            }
        ]
    },
    "settings": {
        "react": {
            "version": "detect"
        },
        "import/resolver": {
            "typescript": true,
            "node": true
        }
    }
}
