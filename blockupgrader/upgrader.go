package blockupgrader

// Block holds the data that identifies a block. It is implemented by BlockState
// and BlockMeta.
type Block interface {
	upgrade() BlockState
}

// Upgrade upgrades the given block using the registered block state upgrade
// schemas. If a Block has not been changed through several versions, Upgrade
// will simply return the original value. Calling blockupgrader.Upgrade is
// therefore safe regardless of whether the block is already up-to-date or not.
func Upgrade(b Block) BlockState {
	return b.upgrade()
}

// BlockMeta holds the name and metadata value (int16) of a block. This format
// is used by blocks from Minecraft Bedrock Edition versions before v1.13.
type BlockMeta struct {
	Name     string `nbt:"name"`
	Metadata int16  `nbt:"meta"`
}

// upgrade is not currently implemented. It panics when called.
func (b BlockMeta) upgrade() BlockState {
	panic("BlockMeta.upgrade: not currently implemented")
}

// BlockState holds the name, properties and version of a block. The name and
// properties of the same block may differ, depending on the Version.
type BlockState struct {
	Name       string         `nbt:"name"`
	Properties map[string]any `nbt:"states"`
	Version    int32          `nbt:"version"`
}

// upgrade upgrades a BlockState to a new BlockState, changing its Name,
// Properties and Version if necessary.
func (state BlockState) upgrade() BlockState {
	version := state.Version
	for _, s := range schemas {
		resVersion := s.id
		if version > resVersion {
			continue
		}

		oldName := state.Name
		oldProperties := state.Properties
		if _, ok := s.remappedStates[oldName]; ok {
			var nextSchema bool
			for _, remap := range s.remappedStates[oldName] {
				if len(oldProperties) != len(remap.oldProperties) {
					continue
				}

				var nextState bool
				for k, v := range oldProperties {
					if _, ok := remap.oldProperties[k]; !ok {
						nextState = true
						break
					}
					if remap.oldProperties[k] != v {
						nextState = true
						break
					}
				}
				if nextState {
					continue
				}

				state, nextSchema = BlockState{
					Name:       remap.newName,
					Properties: remap.newProperties,
					Version:    resVersion,
				}, true
				break
			}
			if nextSchema {
				continue
			}
		}

		name, nameRenamed := s.renamedIDs[oldName]
		if !nameRenamed {
			name = oldName
		}

		properties := state.Properties
		propertyAdded := s.applyPropertyAdded(oldName, properties)
		propertyRemoved := s.applyPropertyRemoved(oldName, properties)
		propertyRenamedOrValueChanged := s.applyPropertyRenamedOrValueChanged(oldName, properties)
		propertyValueChanged := s.applyPropertyValueChanged(oldName, properties)

		if nameRenamed || propertyAdded || propertyRemoved || propertyRenamedOrValueChanged || propertyValueChanged {
			state = BlockState{
				Name:       name,
				Properties: properties,
				Version:    resVersion,
			}
		}
	}
	return state
}
