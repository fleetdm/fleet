package msi

// schema.go declares the schemas of every table in the fleetd MSI. The
// column type words and the _Validation metadata are transcribed verbatim
// from WiX 3.14 output so that the generated _Columns and _Validation
// streams match WiX's byte for byte.

// column describes one column of a table: its raw _Columns type word and its
// _Validation row. The type word encodes string/int/nullable/key/length bits
// exactly as stored in the _Columns stream.
type column struct {
	name string
	typ  uint16

	// _Validation row fields (Nullable, MinValue, MaxValue, KeyTable,
	// KeyColumn, Category, Set, Description). Min/Max/KeyColumn use int
	// pointers because 0 and null are distinct.
	nullable    string
	minValue    *int32
	maxValue    *int32
	keyTable    string
	keyColumn   *int16
	category    string
	set         string
	description string
}

func (c column) isString() bool { return c.typ&0x0800 != 0 }

// isObject reports whether the column holds a binary stream (the Binary
// table's Data column): typed as a string column but its cells carry a raw
// stream index rather than a string-pool ID.
func (c column) isObject() bool { return c.category == "Binary" }
func (c column) isKey() bool    { return c.typ&0x2000 != 0 }
func (c column) cellBytes() int {
	if c.isString() || c.typ&0xFF != 4 {
		return 2
	}
	return 4
}

type tableSchema struct {
	name string
	cols []column
}

func i32(v int32) *int32 { return new(v) }
func i16(v int16) *int16 { return new(v) }

// fleetdSchemas lists every table of the fleetd MSI in the order WiX
// processes tables (invariant-culture sort: '_' before letters). The
// _SummaryInformation entry only contributes _Validation rows (its data
// lives in the \x05SummaryInformation stream); Signature has no rows but its
// (empty) table must exist for the RegLocator foreign keys.
var fleetdSchemas = []tableSchema{
	{name: "_SummaryInformation", cols: []column{
		{name: "PropertyId", typ: 0xa502, nullable: "N"},
		{name: "Value", typ: 0x8fff, nullable: "N"},
	}},
	{name: "_Validation", cols: []column{
		{name: "Table", typ: 0xad20, nullable: "N", category: "Identifier", description: "Name of table"},
		{name: "Column", typ: 0xad20, nullable: "N", category: "Identifier", description: "Name of column"},
		{name: "Nullable", typ: 0x8d04, nullable: "N", set: "Y;N", description: "Whether the column is nullable"},
		{name: "MinValue", typ: 0x9104, nullable: "Y", minValue: i32(-2147483647), maxValue: i32(2147483647), description: "Minimum value allowed"},
		{name: "MaxValue", typ: 0x9104, nullable: "Y", minValue: i32(-2147483647), maxValue: i32(2147483647), description: "Maximum value allowed"},
		{name: "KeyTable", typ: 0x9dff, nullable: "Y", category: "Identifier", description: "For foreign key, Name of table to which data must link"},
		{name: "KeyColumn", typ: 0x9502, nullable: "Y", minValue: i32(1), maxValue: i32(32), description: "Column to which foreign key connects"},
		{name: "Category", typ: 0x9d20, nullable: "Y", set: "Text;Formatted;Template;Condition;Guid;Path;Version;Language;Identifier;Binary;UpperCase;LowerCase;Filename;Paths;AnyPath;WildCardFilename;RegPath;CustomSource;Property;Cabinet;Shortcut;FormattedSDDLText;Integer;DoubleInteger;TimeDate;DefaultDir", description: "String category"},
		{name: "Set", typ: 0x9dff, nullable: "Y", category: "Text", description: "Set of values that are permitted"},
		{name: "Description", typ: 0x9dff, nullable: "Y", category: "Text", description: "Description of column"},
	}},
	{name: "AdminExecuteSequence", cols: sequenceColumns()},
	{name: "AdminUISequence", cols: sequenceColumns()},
	{name: "AdvtExecuteSequence", cols: sequenceColumns()},
	{name: "AppSearch", cols: []column{
		{name: "Property", typ: 0xad48, nullable: "N", category: "Identifier", description: "The property associated with a Signature"},
		{name: "Signature_", typ: 0xad48, nullable: "N", keyTable: "Signature;RegLocator;IniLocator;DrLocator;CompLocator", keyColumn: i16(1), category: "Identifier", description: "The Signature_ represents a unique file signature and is also the foreign key in the Signature,  RegLocator, IniLocator, CompLocator and the DrLocator tables."},
	}},
	{name: "Property", cols: []column{
		{name: "Property", typ: 0xad48, nullable: "N", category: "Identifier", description: "Name of property, uppercase if settable by launcher or loader."},
		{name: "Value", typ: 0x8f00, nullable: "N", category: "Text", description: "String value for property.  Never null or empty."},
	}},
	{name: "Binary", cols: []column{
		{name: "Name", typ: 0xad48, nullable: "N", category: "Identifier", description: "Unique key identifying the binary data."},
		{name: "Data", typ: 0x8900, nullable: "N", category: "Binary", description: "The unformatted binary data."},
	}},
	{name: "Component", cols: []column{
		{name: "Component", typ: 0xad48, nullable: "N", category: "Identifier", description: "Primary key used to identify a particular component record."},
		{name: "ComponentId", typ: 0x9d26, nullable: "Y", category: "Guid", description: "A string GUID unique to this component, version, and language."},
		{name: "Directory_", typ: 0x8d48, nullable: "N", keyTable: "Directory", keyColumn: i16(1), category: "Identifier", description: "Required key of a Directory table record. This is actually a property name whose value contains the actual path, set either by the AppSearch action or with the default setting obtained from the Directory table."},
		{name: "Attributes", typ: 0x8502, nullable: "N", description: "Remote execution option, one of irsEnum"},
		{name: "Condition", typ: 0x9dff, nullable: "Y", category: "Condition", description: "A conditional statement that will disable this component if the specified condition evaluates to the 'True' state. If a component is disabled, it will not be installed, regardless of the 'Action' state associated with the component."},
		{name: "KeyPath", typ: 0x9d48, nullable: "Y", keyTable: "File;Registry;ODBCDataSource", keyColumn: i16(1), category: "Identifier", description: "Either the primary key into the File table, Registry table, or ODBCDataSource table. This extract path is stored when the component is installed, and is used to detect the presence of the component and to return the path to it."},
	}},
	{name: "Directory", cols: []column{
		{name: "Directory", typ: 0xad48, nullable: "N", category: "Identifier", description: "Unique identifier for directory entry, primary key. If a property by this name is defined, it contains the full path to the directory."},
		{name: "Directory_Parent", typ: 0x9d48, nullable: "Y", keyTable: "Directory", keyColumn: i16(1), category: "Identifier", description: "Reference to the entry in this table specifying the default parent directory. A record parented to itself or with a Null parent represents a root of the install tree."},
		{name: "DefaultDir", typ: 0x8fff, nullable: "N", category: "DefaultDir", description: "The default sub-path under parent's path."},
	}},
	{name: "CreateFolder", cols: []column{
		{name: "Directory_", typ: 0xad48, nullable: "N", keyTable: "Directory", keyColumn: i16(1), category: "Identifier", description: "Primary key, could be foreign key into the Directory table."},
		{name: "Component_", typ: 0xad48, nullable: "N", keyTable: "Component", keyColumn: i16(1), category: "Identifier", description: "Foreign key into the Component table."},
	}},
	{name: "CustomAction", cols: []column{
		{name: "Action", typ: 0xad48, nullable: "N", category: "Identifier", description: "Primary key, name of action, normally appears in sequence table unless private use."},
		{name: "Type", typ: 0x8502, nullable: "N", minValue: i32(1), maxValue: i32(32767), description: "The numeric custom action type, consisting of source location, code type, entry, option flags."},
		{name: "Source", typ: 0x9d48, nullable: "Y", category: "CustomSource", description: "The table reference of the source of the code."},
		{name: "Target", typ: 0x9dff, nullable: "Y", category: "Formatted", description: "Excecution parameter, depends on the type of custom action"},
		{name: "ExtendedType", typ: 0x9104, nullable: "Y", minValue: i32(0), maxValue: i32(2147483647), description: "A numeric custom action type that extends code type or option flags of the Type column."},
	}},
	{name: "Environment", cols: []column{
		{name: "Environment", typ: 0xad48, nullable: "N", category: "Identifier", description: "Unique identifier for the environmental variable setting"},
		{name: "Name", typ: 0x8fff, nullable: "N", category: "Text", description: "The name of the environmental value."},
		{name: "Value", typ: 0x9fff, nullable: "Y", category: "Formatted", description: "The value to set in the environmental settings."},
		{name: "Component_", typ: 0x8d48, nullable: "N", keyTable: "Component", keyColumn: i16(1), category: "Identifier", description: "Foreign key into the Component table referencing component that controls the installing of the environmental value."},
	}},
	{name: "Feature", cols: []column{
		{name: "Feature", typ: 0xad26, nullable: "N", category: "Identifier", description: "Primary key used to identify a particular feature record."},
		{name: "Feature_Parent", typ: 0x9d26, nullable: "Y", keyTable: "Feature", keyColumn: i16(1), category: "Identifier", description: "Optional key of a parent record in the same table. If the parent is not selected, then the record will not be installed. Null indicates a root item."},
		{name: "Title", typ: 0x9f40, nullable: "Y", category: "Text", description: "Short text identifying a visible feature item."},
		{name: "Description", typ: 0x9fff, nullable: "Y", category: "Text", description: "Longer descriptive text describing a visible feature item."},
		{name: "Display", typ: 0x9502, nullable: "Y", minValue: i32(0), maxValue: i32(32767), description: "Numeric sort order, used to force a specific display ordering."},
		{name: "Level", typ: 0x8502, nullable: "N", minValue: i32(0), maxValue: i32(32767), description: "The install level at which record will be initially selected. An install level of 0 will disable an item and prevent its display."},
		{name: "Directory_", typ: 0x9d48, nullable: "Y", keyTable: "Directory", keyColumn: i16(1), category: "UpperCase", description: "The name of the Directory that can be configured by the UI. A non-null value will enable the browse button."},
		{name: "Attributes", typ: 0x8502, nullable: "N", set: "0;1;2;4;5;6;8;9;10;16;17;18;20;21;22;24;25;26;32;33;34;36;37;38;48;49;50;52;53;54", description: "Feature attributes"},
	}},
	{name: "FeatureComponents", cols: []column{
		{name: "Feature_", typ: 0xad26, nullable: "N", keyTable: "Feature", keyColumn: i16(1), category: "Identifier", description: "Foreign key into Feature table."},
		{name: "Component_", typ: 0xad48, nullable: "N", keyTable: "Component", keyColumn: i16(1), category: "Identifier", description: "Foreign key into Component table."},
	}},
	{name: "File", cols: []column{
		{name: "File", typ: 0xad48, nullable: "N", category: "Identifier", description: "Primary key, non-localized token, must match identifier in cabinet.  For uncompressed files, this field is ignored."},
		{name: "Component_", typ: 0x8d48, nullable: "N", keyTable: "Component", keyColumn: i16(1), category: "Identifier", description: "Foreign key referencing Component that controls the file."},
		{name: "FileName", typ: 0x8fff, nullable: "N", category: "Filename", description: "File name used for installation, may be localized.  This may contain a \"short name|long name\" pair."},
		{name: "FileSize", typ: 0x8104, nullable: "N", minValue: i32(0), maxValue: i32(2147483647), description: "Size of file in bytes (long integer)."},
		{name: "Version", typ: 0x9d48, nullable: "Y", keyTable: "File", keyColumn: i16(1), category: "Version", description: "Version string for versioned files;  Blank for unversioned files."},
		{name: "Language", typ: 0x9d14, nullable: "Y", category: "Language", description: "List of decimal language Ids, comma-separated if more than one."},
		{name: "Attributes", typ: 0x9502, nullable: "Y", minValue: i32(0), maxValue: i32(32767), description: "Integer containing bit flags representing file attributes (with the decimal value of each bit position in parentheses)"},
		{name: "Sequence", typ: 0x8104, nullable: "N", minValue: i32(1), maxValue: i32(2147483647), description: "Sequence with respect to the media images; order must track cabinet order."},
	}},
	{name: "InstallExecuteSequence", cols: sequenceColumns()},
	{name: "InstallUISequence", cols: sequenceColumns()},
	{name: "Media", cols: []column{
		{name: "DiskId", typ: 0xa502, nullable: "N", minValue: i32(1), maxValue: i32(32767), description: "Primary key, integer to determine sort order for table."},
		{name: "LastSequence", typ: 0x8104, nullable: "N", minValue: i32(0), maxValue: i32(2147483647), description: "File sequence number for the last file for this media."},
		{name: "DiskPrompt", typ: 0x9f40, nullable: "Y", category: "Text", description: "Disk name: the visible text actually printed on the disk.  This will be used to prompt the user when this disk needs to be inserted."},
		{name: "Cabinet", typ: 0x9dff, nullable: "Y", category: "Cabinet", description: "If some or all of the files stored on the media are compressed in a cabinet, the name of that cabinet."},
		{name: "VolumeLabel", typ: 0x9d20, nullable: "Y", category: "Text", description: "The label attributed to the volume."},
		{name: "Source", typ: 0x9d48, nullable: "Y", category: "Property", description: "The property defining the location of the cabinet file."},
	}},
	{name: "MsiFileHash", cols: []column{
		{name: "File_", typ: 0xad48, nullable: "N", keyTable: "File", keyColumn: i16(1), category: "Identifier", description: "Primary key, foreign key into File table referencing file with this hash"},
		{name: "Options", typ: 0x8502, nullable: "N", minValue: i32(0), maxValue: i32(32767), description: "Various options and attributes for this hash."},
		{name: "HashPart1", typ: 0x8104, nullable: "N", description: "Size of file in bytes (long integer)."},
		{name: "HashPart2", typ: 0x8104, nullable: "N", description: "Size of file in bytes (long integer)."},
		{name: "HashPart3", typ: 0x8104, nullable: "N", description: "Size of file in bytes (long integer)."},
		{name: "HashPart4", typ: 0x8104, nullable: "N", description: "Size of file in bytes (long integer)."},
	}},
	{name: "MsiLockPermissionsEx", cols: []column{
		{name: "MsiLockPermissionsEx", typ: 0xad48, nullable: "N", category: "Identifier", description: "Primary key, non-localized token"},
		{name: "LockObject", typ: 0x8d48, nullable: "N", category: "Identifier", description: "Foreign key into Registry, File, CreateFolder, or ServiceInstall table"},
		{name: "Table", typ: 0x8d20, nullable: "N", category: "Identifier", set: "CreateFolder;File;Registry;ServiceInstall", description: "Reference to another table name"},
		{name: "SDDLText", typ: 0x8d00, nullable: "N", category: "FormattedSDDLText", description: "String to indicate permissions to be applied to the LockObject"},
		{name: "Condition", typ: 0x9dff, nullable: "Y", category: "Formatted", description: "Expression which must evaluate to TRUE in order for this set of permissions to be applied"},
	}},
	{name: "RegLocator", cols: []column{
		{name: "Signature_", typ: 0xad48, nullable: "N", category: "Identifier", description: "The table key. The Signature_ represents a unique file signature and is also the foreign key in the Signature table. If the type is 0, the registry values refers a directory, and _Signature is not a foreign key."},
		{name: "Root", typ: 0x8502, nullable: "N", minValue: i32(0), maxValue: i32(3), description: "The predefined root key for the registry value, one of rrkEnum."},
		{name: "Key", typ: 0x8dff, nullable: "N", category: "RegPath", description: "The key for the registry value."},
		{name: "Name", typ: 0x9dff, nullable: "Y", category: "Formatted", description: "The registry value name."},
		{name: "Type", typ: 0x9502, nullable: "Y", minValue: i32(0), maxValue: i32(18), description: "An integer value that determines if the registry value is a filename or a directory location or to be used as is w/o interpretation."},
	}},
	{name: "ServiceConfig", cols: []column{
		{name: "ServiceName", typ: 0xad48, nullable: "N", category: "Formatted", description: "Primary key, non-localized token"},
		{name: "Component_", typ: 0x8d48, nullable: "N", keyTable: "Component", keyColumn: i16(1), category: "Identifier", description: "Foreign key, Component used to determine install state "},
		{name: "NewService", typ: 0x8502, nullable: "N", minValue: i32(0), maxValue: i32(1), description: "Whether the affected service is being installed or already exists."},
		{name: "FirstFailureActionType", typ: 0x8d20, nullable: "N", category: "Text", description: "First failure action type for configured service to take."},
		{name: "SecondFailureActionType", typ: 0x8d20, nullable: "N", category: "Text", description: "Second failure action type for configured service to take."},
		{name: "ThirdFailureActionType", typ: 0x8d20, nullable: "N", category: "Text", description: "Third failure action type for configured service to take."},
		{name: "ResetPeriodInDays", typ: 0x9104, nullable: "Y", minValue: i32(0), category: "Integer", description: "Period after which to reset the failure count for the service."},
		{name: "RestartServiceDelayInSeconds", typ: 0x9104, nullable: "Y", minValue: i32(0), category: "Integer", description: "Period after which to restart the service after a given failure."},
		{name: "ProgramCommandLine", typ: 0x9dff, nullable: "Y", category: "Formatted", description: "Command line for program to run if failure action is RUN_COMMAND."},
		{name: "RebootMessage", typ: 0x9dff, nullable: "Y", category: "Text", description: "Message to show to users when rebooting if failure action is REBOOT."},
	}},
	{name: "ServiceControl", cols: []column{
		{name: "ServiceControl", typ: 0xad48, nullable: "N", category: "Identifier", description: "Primary key, non-localized token."},
		{name: "Name", typ: 0x8fff, nullable: "N", category: "Formatted", description: "Name of a service. /, \\, comma and space are invalid"},
		{name: "Event", typ: 0x8502, nullable: "N", minValue: i32(0), maxValue: i32(187), description: "Bit field:  Install:  0x1 = Start, 0x2 = Stop, 0x8 = Delete, Uninstall: 0x10 = Start, 0x20 = Stop, 0x80 = Delete"},
		{name: "Arguments", typ: 0x9fff, nullable: "Y", category: "Formatted", description: "Arguments for the service.  Separate by [~]."},
		{name: "Wait", typ: 0x9502, nullable: "Y", minValue: i32(0), maxValue: i32(1), description: "Boolean for whether to wait for the service to fully start"},
		{name: "Component_", typ: 0x8d48, nullable: "N", keyTable: "Component", keyColumn: i16(1), category: "Identifier", description: "Required foreign key into the Component Table that controls the startup of the service"},
	}},
	{name: "ServiceInstall", cols: []column{
		{name: "ServiceInstall", typ: 0xad48, nullable: "N", category: "Identifier", description: "Primary key, non-localized token."},
		{name: "Name", typ: 0x8dff, nullable: "N", category: "Formatted", description: "Internal Name of the Service"},
		{name: "DisplayName", typ: 0x9fff, nullable: "Y", category: "Formatted", description: "External Name of the Service"},
		{name: "ServiceType", typ: 0x8104, nullable: "N", minValue: i32(-2147483647), maxValue: i32(2147483647), description: "Type of the service"},
		{name: "StartType", typ: 0x8104, nullable: "N", minValue: i32(0), maxValue: i32(4), description: "Type of the service"},
		{name: "ErrorControl", typ: 0x8104, nullable: "N", minValue: i32(-2147483647), maxValue: i32(2147483647), description: "Severity of error if service fails to start"},
		{name: "LoadOrderGroup", typ: 0x9dff, nullable: "Y", category: "Formatted", description: "LoadOrderGroup"},
		{name: "Dependencies", typ: 0x9dff, nullable: "Y", category: "Formatted", description: "Other services this depends on to start.  Separate by [~], and end with [~][~]"},
		{name: "StartName", typ: 0x9dff, nullable: "Y", category: "Formatted", description: "User or object name to run service as"},
		{name: "Password", typ: 0x9dff, nullable: "Y", category: "Formatted", description: "password to run service with.  (with StartName)"},
		{name: "Arguments", typ: 0x9dff, nullable: "Y", category: "Formatted", description: "Arguments to include in every start of the service, passed to WinMain"},
		{name: "Component_", typ: 0x8d48, nullable: "N", keyTable: "Component", keyColumn: i16(1), category: "Identifier", description: "Required foreign key into the Component Table that controls the startup of the service"},
		{name: "Description", typ: 0x9fff, nullable: "Y", category: "Text", description: "Description of service."},
	}},
	{name: "Signature", cols: []column{
		{name: "Signature", typ: 0xad48, nullable: "N", category: "Identifier", description: "The table key. The Signature represents a unique file signature."},
		{name: "FileName", typ: 0x8dff, nullable: "N", category: "Text", description: "The name of the file. This may contain a \"short name|long name\" pair."},
		{name: "MinVersion", typ: 0x9d14, nullable: "Y", category: "Text", description: "The minimum version of the file."},
		{name: "MaxVersion", typ: 0x9d14, nullable: "Y", category: "Text", description: "The maximum version of the file."},
		{name: "MinSize", typ: 0x9104, nullable: "Y", minValue: i32(0), maxValue: i32(2147483647), description: "The minimum size of the file."},
		{name: "MaxSize", typ: 0x9104, nullable: "Y", minValue: i32(0), maxValue: i32(2147483647), description: "The maximum size of the file. "},
		{name: "MinDate", typ: 0x9104, nullable: "Y", minValue: i32(0), maxValue: i32(2147483647), description: "The minimum creation date of the file."},
		{name: "MaxDate", typ: 0x9104, nullable: "Y", minValue: i32(0), maxValue: i32(2147483647), description: "The maximum creation date of the file."},
		{name: "Languages", typ: 0x9dff, nullable: "Y", category: "Language", description: "The languages supported by the file."},
	}},
	{name: "Upgrade", cols: []column{
		{name: "UpgradeCode", typ: 0xad26, nullable: "N", category: "Guid", description: "The UpgradeCode GUID belonging to the products in this set."},
		{name: "VersionMin", typ: 0xbd14, nullable: "Y", category: "Text", description: "The minimum ProductVersion of the products in this set.  The set may or may not include products with this particular version."},
		{name: "VersionMax", typ: 0xbd14, nullable: "Y", category: "Text", description: "The maximum ProductVersion of the products in this set.  The set may or may not include products with this particular version."},
		{name: "Language", typ: 0xbdff, nullable: "Y", category: "Language", description: "A comma-separated list of languages for either products in this set or products not in this set."},
		{name: "Attributes", typ: 0xa104, nullable: "N", minValue: i32(0), maxValue: i32(2147483647), description: "The attributes of this product set."},
		{name: "Remove", typ: 0x9dff, nullable: "Y", category: "Formatted", description: "The list of features to remove when uninstalling a product from this set.  The default is \"ALL\"."},
		{name: "ActionProperty", typ: 0x8d48, nullable: "N", category: "UpperCase", description: "The property to set when a product in this set is found."},
	}},
}

// sequenceColumns returns the shared schema of the five sequence tables.
func sequenceColumns() []column {
	return []column{
		{name: "Action", typ: 0xad48, nullable: "N", category: "Identifier", description: "Name of action to invoke, either in the engine or the handler DLL."},
		{name: "Condition", typ: 0x9dff, nullable: "Y", category: "Condition", description: "Optional expression which skips the action if evaluates to expFalse.If the expression syntax is invalid, the engine will terminate, returning iesBadActionData."},
		{name: "Sequence", typ: 0x9502, nullable: "Y", minValue: i32(-4), maxValue: i32(32767), description: "Number that determines the sort order in which the actions are to be executed.  Leave blank to suppress action."},
	}
}
