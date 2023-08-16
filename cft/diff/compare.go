package diff

import (
	"math"
	"reflect"

	"github.com/elishacatherasoo/rain/cft"
)

// New returns a Diff that represents the difference between two templates
func New(a, b cft.Template) Diff {
	return compareMaps(a.Map(), b.Map())
}

func compareValues(old, new interface{}) Diff {
	if reflect.TypeOf(old) != reflect.TypeOf(new) {
		return value{new, Changed}
	}

	switch v := old.(type) {
	case []interface{}:
		return compareSlices(v, new.([]interface{}))
	case map[string]interface{}:
		return compareMaps(v, new.(map[string]interface{}))
	default:
		if !reflect.DeepEqual(old, new) {
			return value{new, Changed}
		}
	}

	return value{old, Unchanged}
}

func compareSlices(old, new []interface{}) Diff {
	max := int(math.Max(float64(len(old)), float64(len(new))))
	d := make(slice, max)

	for i := 0; i < max; i++ {
		if i >= len(old) {
			d[i] = value{new[i], Added}
		} else if i >= len(new) {
			d[i] = value{old[i], Removed}
		} else {
			d[i] = compareValues(old[i], new[i])
		}
	}

	return d
}

func compareMaps(old, new map[string]interface{}) Diff {
	d := make(dmap)

	// New and updated keys
	for key, val := range new {
		if _, ok := old[key]; !ok {
			d[key] = value{val, Added}
		} else {
			d[key] = compareValues(old[key], val)
		}
	}

	// Removed keys
	for key, val := range old {
		if _, ok := new[key]; !ok {
			d[key] = value{val, Removed}
		}
	}

	return d
}
