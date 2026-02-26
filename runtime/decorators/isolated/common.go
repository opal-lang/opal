package isolated

// Path identifies a decorator path under the isolated namespace.
type Path string

// Namespace is the canonical root for isolation decorators.
const Namespace Path = "isolated"

// FullPath joins namespace with a relative sub-path.
func FullPath(subpath string) string {
	if subpath == "" {
		return string(Namespace)
	}

	return string(Namespace) + "." + subpath
}
