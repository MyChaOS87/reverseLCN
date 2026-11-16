package monitor

import "maps"

// Option configures a DataStore instance.
type Option func(*DataStore)

// WithIDMap configures optional module ID to label mappings.
func WithIDMap(idMap map[int]string) Option {
	return func(d *DataStore) {
		if len(idMap) > 0 {
			d.idMap = maps.Clone(idMap)
		}
	}
}

// WithModuleOutputs configures optional per-module output channel label mappings.
func WithModuleOutputs(moduleOutputs map[int]map[int]string) Option {
	return func(d *DataStore) {
		if len(moduleOutputs) == 0 {
			return
		}

		d.moduleOutputs = make(map[int]map[int]string, len(moduleOutputs))
		for mod, outs := range moduleOutputs {
			d.moduleOutputs[mod] = maps.Clone(outs)
		}
	}
}
