# JSON Field Selection Example

This example demonstrates how to use the `@json` directive to select nested JSON fields from JSONB columns in PostgreSQL.

## Setup

1. **Start PostgreSQL** (if not already running):
   ```bash
   docker run -d --name postgres-json-example \
     -e POSTGRES_PASSWORD=postgres \
     -e POSTGRES_DB=postgres \
     -p 5432:5432 \
     postgres:15
   ```

2. **Initialize the database**:
   ```bash
   psql -h localhost -U postgres -d postgres -f init.sql
   ```

   Or if using a password:
   ```bash
   PGPASSWORD=postgres psql -h localhost -U postgres -d postgres -f init.sql
   ```

3. **Generate GraphQL code**:
   ```bash
   cd examples/json
   go generate
   ```

4. **Run the server**:
   ```bash
   go run server.go
   ```

   Or with custom connection string:
   ```bash
   PG_CONN_STR="postgresql://localhost/postgres?user=postgres&password=postgres" go run server.go
   ```

5. **Open GraphQL Playground**:
   Navigate to http://localhost:8080/

## Test Queries

### Simple Scalar Selection
```graphql
query {
  products {
    name
    attributes {
      color
      size
    }
  }
}
```

### Nested Object Selection
```graphql
query {
  products {
    name
    attributes {
      color
      details {
        manufacturer
        model
      }
    }
  }
}
```

### Deep Nesting (3 levels)
```graphql
query {
  products {
    name
    attributes {
      details {
        warranty {
          years
          provider
        }
      }
    }
  }
}
```

### Three-Level Nesting with Dimensions
```graphql
query {
  products {
    name
    attributes {
      specs {
        dimensions {
          width
          height
          depth
        }
      }
    }
  }
}
```

### Mixed Scalar and Nested
```graphql
query {
  products {
    name
    attributes {
      color
      size
      details {
        manufacturer
      }
      specs {
        weight
      }
    }
  }
}
```

### Complex Nested Object with All Fields
```graphql
query {
  products {
    name
    attributes {
      details {
        manufacturer
        model
        warranty {
          years
          provider
        }
      }
    }
  }
}
```

## Expected Results

- **Product 1**: Simple attributes (color, size)
- **Product 2**: Attributes with nested details (manufacturer, model)
- **Product 3**: Deep nesting with warranty info
- **Product 4**: Three-level nesting with dimensions
- **Product 5**: Complex nested structure with all fields

## Notes

- The `@json` directive tells FastGQL that the field is stored in a JSONB column
- Nested field selection uses efficient PostgreSQL `->` operators
- The generated SQL uses `jsonb_build_object` to construct the response matching the GraphQL query structure

