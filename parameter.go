package cloudcostexplorer

import "strings"

// Parameter is the configuration of a filtering and/or grouping parameter available for the cloud service.
type Parameter struct {
	ID              string // parameter ID, like "SERVICE".
	Name            string // parameter name, like "Service".
	MenuTitle       string // if set, this will be shown in menus instead of Name.
	DefaultPriority int    // if > 0, sets a priority to select a group if the previous group contains a single item.
	IsGroup         bool   // set whether the parameter can be used for grouping.
	IsGroupFilter   bool   // if IsGroup==true, sets whether the field value will have a link to filter by its value.
	IsFilter        bool   // sets whether the parameter can be used for filtering.
	HasData         bool   // sets whether the parameter may contain extra data in its name, separated by DataSeparator.
	DataRequired    bool   // if HasData==true, sets whether the parameter always contains data or only sometimes.
}

type Parameters []Parameter

// FindById finds a parameter by ID. If the parameter was not found and has data, it is split by DataSeparator and
// searched for its first part.
func (p Parameters) FindById(id string) (Parameter, bool) {
	for _, parameter := range p {
		if parameter.ID == id {
			return parameter, true
		}
		if parameter.HasData {
			dn, _, ok := strings.Cut(id, DataSeparator)
			if !ok {
				continue
			}
			if parameter.ID == dn {
				return parameter, true
			}
		}
	}
	return Parameter{}, false
}

// FindByGroupDefaultPriority finds a parameter which is a group and has the passed default priority.
func (p Parameters) FindByGroupDefaultPriority(priority int) (Parameter, bool) {
	for _, parameter := range p {
		if parameter.IsGroup && parameter.DefaultPriority == priority {
			return parameter, true
		}
	}
	return Parameter{}, false
}

// DefaultGroup returns the parameter which is a group that should be the default group if none was selected.
func (p Parameters) DefaultGroup() Parameter {
	var firstGroup *Parameter
	for _, parameter := range p {
		if parameter.IsGroup && parameter.DefaultPriority == 1 {
			return parameter
		}
		if firstGroup == nil && parameter.IsGroup {
			firstGroup = &parameter
		}
	}
	if firstGroup != nil {
		return *firstGroup
	}
	return Parameter{}
}
