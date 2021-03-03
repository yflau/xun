package dbal

// Grammars loaded grammar driver
var Grammars = map[string]Grammar{}

// Register register the grammar driver
func Register(name string, grammar Grammar) {
	Grammars[name] = grammar
}

// GetName get the table name
func (table *Table) GetName() string {
	return table.TableName
}

// NewTable create a grammar table
func NewTable(name string, schemaName string, dbName string) *Table {
	return &Table{
		DBName:     dbName,
		SchemaName: schemaName,
		TableName:  name,
		Primary:    nil,
		Columns:    []*Column{},
		ColumnMap:  map[string]*Column{},
		Indexes:    []*Index{},
		IndexMap:   map[string]*Index{},
		Commands:   []*Command{},
	}
}

// NewPrimary create a new primary intstance
func (table *Table) NewPrimary(name string, columns ...*Column) *Primary {
	return &Primary{
		DBName:    table.DBName,
		TableName: table.TableName,
		Table:     table,
		Name:      name,
		Columns:   columns,
	}
}

// GetPrimary  get the primary key instance
func (table *Table) GetPrimary(name string, columns ...*Column) *Primary {
	return table.Primary
}

// NewColumn create a new column intstance
func (table *Table) NewColumn(name string) *Column {
	return &Column{
		DBName:            table.DBName,
		TableName:         table.TableName,
		Table:             table,
		Name:              name,
		Length:            nil,
		OctetLength:       nil,
		Precision:         nil,
		Scale:             nil,
		DatetimePrecision: nil,
		Charset:           nil,
		Collation:         nil,
		Key:               nil,
		Extra:             nil,
		Comment:           nil,
	}
}

// PushColumn push a column instance to the table columns
func (table *Table) PushColumn(column *Column) *Table {
	table.ColumnMap[column.Name] = column
	table.Columns = append(table.Columns, column)
	return table
}

// HasColumn checking if the given name column exists
func (table *Table) HasColumn(name string) bool {
	_, has := table.ColumnMap[name]
	return has
}

// GetColumn get the given name column instance
func (table *Table) GetColumn(name string) *Column {
	return table.ColumnMap[name]
}

// NewIndex create a new index intstance
func (table *Table) NewIndex(name string, columns ...*Column) *Index {
	return &Index{
		DBName:    table.DBName,
		TableName: table.TableName,
		Table:     table,
		Name:      name,
		Columns:   columns,
	}
}

// PushIndex push an index instance to the table indexes
func (table *Table) PushIndex(index *Index) *Table {
	table.IndexMap[index.Name] = index
	table.Indexes = append(table.Indexes, index)
	return table
}

// HasIndex checking if the given name index exists
func (table *Table) HasIndex(name string) bool {
	_, has := table.IndexMap[name]
	return has
}

// GetIndex get the given name index instance
func (table *Table) GetIndex(name string) *Index {
	return table.IndexMap[name]
}

// AddCommand Add a new command to the table.
//
// The commands must be:
//    AddColumn(column *Column)    for adding a column
//    ModifyColumn(column *Column) for modifying a colu
//    RenameColumn(old string,new string)  for renaming a column
//    DropColumn(name string)  for dropping a column
//    CreateIndex(index *Index) for creating a index
//    DropIndex( name string) for  dropping a index
//    RenameIndex(old string,new string)  for renaming a index
func (table *Table) AddCommand(name string, success func(), fail func(), params ...interface{}) {
	table.Commands = append(table.Commands, &Command{
		Name:    name,
		Params:  params,
		Success: success,
		Fail:    fail,
	})
}

// Callback run the callback code
func (command *Command) Callback(err error) {
	if err == nil && command.Success != nil {
		command.Success()
	} else if err != nil && command.Fail != nil {
		command.Fail()
	}
}