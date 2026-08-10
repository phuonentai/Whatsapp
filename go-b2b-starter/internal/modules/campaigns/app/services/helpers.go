package services

// toIntArray converts a JSON-decoded value to a []int32 (single number is
// accepted as a one-element array).
func toIntArray(value any) ([]int32, bool) {
	switch v := value.(type) {
	case []int32:
		return v, true
	case []any:
		out := make([]int32, 0, len(v))
		for _, item := range v {
			n, ok := toInt(item)
			if !ok {
				return nil, false
			}
			out = append(out, n)
		}
		return out, true
	}
	return nil, false
}

func toInt(value any) (int32, bool) {
	switch v := value.(type) {
	case float64:
		if v != float64(int32(v)) {
			return 0, false
		}
		return int32(v), true
	case int:
		return int32(v), true
	case int32:
		return v, true
	case int64:
		return int32(v), true
	}
	return 0, false
}
