# FastGQL Documentation

This directory contains the official documentation for FastGQL, built with [Starlight](https://starlight.astro.build) (Astro documentation framework).

The documentation is published at [https://www.fastgql.io](https://www.fastgql.io)

## 📁 Project Structure

```
docs/
├── public/              # Static assets (favicons, images, etc.)
├── src/
│   ├── assets/         # Images and other assets used in documentation
│   ├── content/
│   │   ├── docs/       # Documentation markdown files
│   │   │   ├── start/  # Getting started guides
│   │   │   ├── schema/ # Schema and directives documentation
│   │   │   ├── queries/# Query documentation
│   │   │   └── reference/ # API reference
│   │   └── config.ts   # Documentation configuration
│   └── env.d.ts
├── astro.config.mjs    # Astro configuration
├── package.json
└── tsconfig.json
```

## 🚀 Commands

All commands should be run from the `docs/` directory:

| Command                   | Action                                           |
| :------------------------ | :----------------------------------------------- |
| `npm install`             | Installs dependencies                            |
| `npm run dev`             | Starts local dev server at `localhost:4321`      |
| `npm run build`           | Build your production site to `./dist/`          |
| `npm run preview`         | Preview your build locally, before deploying     |

## 📝 Contributing to Documentation

Documentation files are located in `src/content/docs/` and are written in Markdown (`.md`) or MDX (`.mdx`) format.

Each file is automatically exposed as a route based on its file path. For example:
- `src/content/docs/start/setup.md` → `/start/setup`
- `src/content/docs/schema/directives.md` → `/schema/directives`

### Adding New Pages

1. Create a new `.md` file in the appropriate directory under `src/content/docs/`
2. Add frontmatter with `title` and `description`:
   ```markdown
   ---
   title: Page Title
   description: Page description
   ---
   ```
3. Write your content in Markdown

### Adding Images

Images can be added to `src/assets/` and referenced in Markdown with relative paths.

## 🔗 Links

- [FastGQL Repository](https://github.com/roneli/fastgql)
- [FastGQL Documentation](https://www.fastgql.io)
- [Starlight Documentation](https://starlight.astro.build/)
