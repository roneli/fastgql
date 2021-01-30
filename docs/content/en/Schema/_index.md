---
title: "Schema"
linkTitle: "Schema"
weight: 9
description: >
  Schema, data model and directives
---

With *fastGQL* you can quickly model your data and relations that exist between them via the GraphQL Schema Definition Language (SDL).

Data modelling in *fastGQL* has two significant roles:

- Define the underlying database schema (fastgql maps the types and directives we defined in the SDL to database tables and constraints). 
- gqlgen code (model + server) generation, and fastgql resolver generations 

If you are already excited and want to auto generate boring crud resolvers, follow this guide to enhance your schema.