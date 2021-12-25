package importer

func isZero(v *string) bool {
	return v == nil || *v == ""
}

func zeroOrValue(v *string) string {
	if v == nil {
		return ""
	}

	return *v
}
