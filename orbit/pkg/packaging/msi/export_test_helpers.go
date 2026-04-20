package msi

import "io"

// Test helpers to expose internal CFB writer for testing.

type CfbWriterWrapper = cfbWriter

func NewCFBWriterForTest() *CfbWriterWrapper             { return newCFBWriter() }
func (cw *cfbWriter) AddStreamForTest(n string, d []byte) { cw.addStream(n, d) }
func (cw *cfbWriter) WriteToForTest(w io.Writer) error     { return cw.writeTo(w) }

// Additional exports for external test programs
func MsiEncodeName(name string, isTable bool) string { return msiEncodeName(name, isTable) }
func ColStr(n uint8) uint16   { return colStr(n) }
func ColStrPK(n uint8) uint16 { return colStrPK(n) }
func ColStrL(n uint8) uint16  { return colStrL(n) }
func ColStrLN(n uint8) uint16 { return colStrLN(n) }
func ColStrN(n uint8) uint16  { return colStrN(n) }

func BuildDatabaseForTest(pool *StringPool, rootDir string, opts MSIOptions) ([]*TableData, []CabFile, error) {
	return buildDatabase(pool, rootDir, opts)
}
