package app

type Column struct {
	Name          string
	DBField       string
	ShowByDefault bool
}

var AvailableColumns = []Column{
	{Name: "Timestamp", DBField: "timestamp", ShowByDefault: true},
	{Name: "ID", DBField: "id", ShowByDefault: true},
	{Name: "Name", DBField: "name", ShowByDefault: true},
	{Name: "Description", DBField: "description", ShowByDefault: false},
	{Name: "Source", DBField: "source", ShowByDefault: false},
	{Name: "Severity", DBField: "severity", ShowByDefault: false},
}

func GetDefaultColumns() []Column {
	defaultColumns := []Column{}
	for _, col := range AvailableColumns {
		if col.ShowByDefault {
			defaultColumns = append(defaultColumns, col)
		}
	}
	return defaultColumns
}

func GetColumnByName(name string) (Column, bool) {
	for _, col := range AvailableColumns {
		if col.Name == name {
			return col, true
		}
	}
	return Column{}, false
}
