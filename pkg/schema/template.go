package schema

import (
	"go/types"
	"reflect"
	"runtime"
	"strings"
)

func ref(p types.Type) string {
	return types.TypeString(p, func(pkg *types.Package) string {
		return pkg.Name()
	})
}

func deref(p types.Type) string {
	return strings.TrimPrefix(ref(p), "*")
}

func getFunctionName(i any) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}
