package runtime

import (
	"path"
	"runtime"
	"strings"
)

// Ancestor returns name and package of closest parent function
// that does not belong to skipped packages.
//
// For example the result could be
//
//	myapp/mypackage.MyType.myFunction
func Ancestor(skipCallers, stackSize int, skipPackages ...string) string {
	p := ""
	pc := make([]uintptr, stackSize)

	runtime.Callers(skipCallers, pc)

	frames := runtime.CallersFrames(pc)

	for {
		frame, more := frames.Next()

		if !more {
			break
		}

		fn := frame.Function

		// Skip unnamed literals.
		if fn == "" || strings.Contains(fn, "{") {
			continue
		}

		parts := strings.Split(fn, "/")
		parts[len(parts)-1] = strings.Split(parts[len(parts)-1], ".")[0]
		p = strings.Join(parts, "/")

		skip := false

		for _, sp := range skipPackages {
			if p == sp {
				skip = true

				break
			}
		}

		if skip {
			continue
		}

		p = path.Base(path.Dir(fn)) + "/" + path.Base(fn)

		break
	}

	return p
}

// CallerFunc returns trimmed path and name of parent function.
func CallerFunc(skip ...int) string {
	skipFrames := 2
	if len(skip) == 1 {
		skipFrames = skip[0]
	}

	pc, _, _, ok := runtime.Caller(skipFrames)
	if !ok {
		return ""
	}

	f := runtime.FuncForPC(pc)

	pathName := path.Base(path.Dir(f.Name())) + "/" + path.Base(f.Name())

	return pathName
}
