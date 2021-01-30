Documentation
====

This directory contains the markdown source files for the static doc site hosted at [fastgql.io](https://fastgql.io)


## Setup

1. Before working with these docs you will need to install hugo, see [Quickstart](https://gohugo.io/getting-started/quick-start/) for instructions.
2. Build and update site's CSS resources:
   ```
   sudo npm install -D autoprefixer
   sudo npm install -D postcss-cli
   sudo npm install -D postcss
   ```
3. Finally, you will need to set up the docsy submodule using the following command: `git submodule update --init --recursive`

## Editing docs

When editing docs run `hugo serve` and a live reload server will start, then navigate to http://localhost:1313 in your browser. Any changes made will be updated in the browser.


## Publishing docs

Docs are hosted using [render.com](https://render.com/) and will be automatically deployed when merging to master.
