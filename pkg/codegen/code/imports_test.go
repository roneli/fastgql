package code

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportPathForDir(t *testing.T) {
	wd, err := os.Getwd()

	require.NoError(t, err)

	assert.Equal(t, "github.com/roneli/fastgql/pkg/codegen/code", ImportPathForDir(wd))
	assert.Equal(t, "github.com/roneli/fastgql/pkg/api", ImportPathForDir(filepath.Join(wd, "..", "..", "api")))

	// doesnt contain go code, but should still give a valid import path
	assert.Equal(t, "github.com/roneli/fastgql/pkg/docs", ImportPathForDir(filepath.Join(wd, "..", "..", "docs")))

	// directory does not exist
	assert.Equal(t, "github.com/roneli/fastgql/pkg/dos", ImportPathForDir(filepath.Join(wd, "..", "..", "dos")))

	// out of module
	assert.Equal(t, "", ImportPathForDir(filepath.Join(wd, "..", "..", "..", "..")))

	if runtime.GOOS == "windows" {
		assert.Equal(t, "", ImportPathForDir("C:/doesnotexist"))
	} else {
		assert.Equal(t, "", ImportPathForDir("/doesnotexist"))
	}
}

func TestNameForDir(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	assert.Equal(t, "tmp", NameForDir("/tmp"))
	assert.Equal(t, "code", NameForDir(wd))
	assert.Equal(t, "codegen", NameForDir(wd+"/.."))
	assert.Equal(t, "pkg", NameForDir(wd+"/../.."))
}