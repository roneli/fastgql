module github.com/roneli/fastgql

go 1.18

require (
	github.com/99designs/gqlgen v0.17.34
	github.com/doug-martin/goqu/v9 v9.10.0
	github.com/iancoleman/strcase v0.1.3
	github.com/jackc/pgx/v4 v4.10.1
	github.com/jinzhu/inflection v1.0.0
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.26.0
	github.com/spf13/afero v1.6.0
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.6.0
	github.com/stretchr/testify v1.8.2
	github.com/vektah/gqlparser/v2 v2.5.6
	golang.org/x/mod v0.10.0 // indirect
	golang.org/x/sys v0.8.0 // indirect
	golang.org/x/tools v0.9.3
)

require (
	github.com/georgysavva/scany/v2 v2.0.0
	github.com/jackc/pgx/v5 v5.4.2
	github.com/json-iterator/go v1.1.12
	go.mongodb.org/mongo-driver v1.9.1
	golang.org/x/text v0.9.0
)

require (
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/gofrs/uuid v4.0.0+incompatible // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.3 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.8.0 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.0.6 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgtype v1.6.2 // indirect
	github.com/jackc/puddle v1.1.3 // indirect
	github.com/jackc/puddle/v2 v2.2.0 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lib/pq v1.10.2 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.0.2 // indirect
	github.com/xdg-go/stringprep v1.0.2 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	golang.org/x/crypto v0.9.0 // indirect
	golang.org/x/sync v0.2.0 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/docker/cli => github.com/docker/cli v20.10.3-0.20221013132413-1d6c6e2367e2+incompatible // 22.06 master branch
	github.com/docker/docker => github.com/docker/docker v20.10.3-0.20221013203545-33ab36d6b304+incompatible // 22.06 branch
	github.com/moby/buildkit => github.com/moby/buildkit v0.10.1-0.20220816171719-55ba9d14360a // same as buildx

	github.com/opencontainers/runc => github.com/opencontainers/runc v1.1.2 // Can be removed on next bump of containerd to > 1.6.4

	// For k8s dependencies, we use a replace directive, to prevent them being
	// upgraded to the version specified in containerd, which is not relevant to the
	// version needed.
	// See https://github.com/docker/buildx/pull/948 for details.
	// https://github.com/docker/buildx/blob/v0.8.1/go.mod#L62-L64
	k8s.io/api => k8s.io/api v0.22.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.4
	k8s.io/client-go => k8s.io/client-go v0.22.4
)
