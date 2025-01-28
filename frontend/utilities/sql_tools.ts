// @ts-ignore
import sqliteParser from "sqlite-parser";
import { intersection, isPlainObject } from "lodash";
import { osqueryTablesAvailable } from "utilities/osquery_tables";
import {
  MACADMINS_EXTENSION_TABLES,
  QUERYABLE_PLATFORMS,
  QueryablePlatform,
} from "interfaces/platform";
import { TableSchemaPlatform } from "interfaces/osquery_table";

type IAstNode = Record<string | number | symbol, unknown>;

// TODO: Research if there are any preexisting types for osquery schema
// TODO: Is it ever possible that osquery_tables.json would be missing name or platforms?
interface IOsqueryTable {
  name: string;
  platforms: TableSchemaPlatform[];
}

type IPlatformDictionary = Record<string, TableSchemaPlatform[]>;

const platformsByTableDictionary: IPlatformDictionary = (osqueryTablesAvailable as IOsqueryTable[]).reduce(
  (dictionary: IPlatformDictionary, osqueryTable) => {
    dictionary[osqueryTable.name] = osqueryTable.platforms;
    return dictionary;
  },
  {}
);

Object.entries(MACADMINS_EXTENSION_TABLES).forEach(([tableName, platforms]) => {
  platformsByTableDictionary[tableName] = platforms;
});

// The isNode and visit functionality is informed by https://lihautan.com/manipulating-ast-with-javascript/#traversing-an-ast
const _isNode = (node: unknown): node is IAstNode => {
  return !!node && isPlainObject(node);
};

const _visit = (
  abstractSyntaxTree: IAstNode,
  callback: (ast: IAstNode) => void
) => {
  if (abstractSyntaxTree) {
    callback(abstractSyntaxTree);

    Object.keys(abstractSyntaxTree).forEach((key) => {
      const childNode = abstractSyntaxTree[key];
      if (Array.isArray(childNode)) {
        childNode.forEach((grandchildNode) => _visit(grandchildNode, callback));
      } else if (childNode && _isNode(childNode)) {
        _visit(childNode, callback);
      }
    });
  }
};

const filterCompatiblePlatforms = (
  sqlTables: string[]
): QueryablePlatform[] => {
  if (!sqlTables.length) {
    return [...QUERYABLE_PLATFORMS]; // if a query has no tables but is still syntatically valid sql, it is treated as compatible with all platforms
  }

  const compatiblePlatforms = intersection(
    ...sqlTables.map(
      (tableName: string) => platformsByTableDictionary[tableName]
    )
  );

  return QUERYABLE_PLATFORMS.filter((p) => compatiblePlatforms.includes(p));
};

export const parseSqlTables = (
  sqlString: string,
  includeCteTables = false
): string[] => {
  let results: string[] = [];

  // Tables defined via common table expression will be excluded from results by default
  const cteTables: string[] = [];

  const _callback = (node: IAstNode) => {
    if (!node) {
      return;
    }

    if (
      (node.variant === "common" || node.variant === "recursive") &&
      node.format === "table" &&
      node.type === "expression"
    ) {
      const targetName = node.target && (node.target as IAstNode).name;
      targetName &&
        typeof targetName === "string" &&
        cteTables.push(targetName);
      return;
    }

    node.variant === "table" &&
      // ignore table-valued functions (see, e.g., https://www.sqlite.org/json1.html#jeach)
      node.type !== "function" &&
      node.name &&
      typeof node.name === "string" &&
      results.push(node.name);
  };

  try {
    const sqlTree = sqliteParser(sqlString);
    _visit(sqlTree, _callback);

    if (cteTables.length && !includeCteTables) {
      results = results.filter((r: string) => !cteTables.includes(r));
    }

    return results;
  } catch (err) {
    // console.log(`sqlite-parser error: ${err}\n\n${sqlString}`);

    throw err;
  }
};

export const checkTable = (
  sqlString = "",
  includeCteTables = false
): { tables: string[] | null; error: Error | null } => {
  let sqlTables: string[] | undefined;
  try {
    sqlTables = parseSqlTables(sqlString, includeCteTables);
  } catch (err) {
    return { tables: null, error: new Error(`${err}`) };
  }

  if (sqlTables === undefined) {
    return {
      tables: null,
      error: new Error(
        "Unexpected error checking table names: sqlTables are undefined"
      ),
    };
  }

  return { tables: sqlTables, error: null };
};

export const checkPlatformCompatibility = (
  sqlString: string,
  includeCteTables = false
): { platforms: QueryablePlatform[] | null; error: Error | null } => {
  let sqlTables: string[] | undefined;
  try {
    // get tables from str
    sqlTables = parseSqlTables(sqlString, includeCteTables);
  } catch (err) {
    return { platforms: null, error: new Error(`${err}`) };
  }

  if (sqlTables === undefined) {
    return {
      platforms: null,
      error: new Error(
        "Unexpected error checking platform compatibility: sqlTables are undefined"
      ),
    };
  }

  try {
    // use tables to get platforms
    const platforms = filterCompatiblePlatforms(sqlTables);
    return { platforms, error: null };
  } catch (err) {
    return { platforms: null, error: new Error(`${err}`) };
  }
};

export const sqlKeyWords = [
  "select",
  "insert",
  "update",
  "delete",
  "from",
  "where",
  "and",
  "or",
  "group",
  "by",
  "order",
  "limit",
  "offset",
  "having",
  "as",
  "case",
  "when",
  "else",
  "end",
  "type",
  "left",
  "right",
  "join",
  "on",
  "outer",
  "desc",
  "asc",
  "union",
  "create",
  "table",
  "primary",
  "key",
  "if",
  "foreign",
  "not",
  "references",
  "default",
  "null",
  "inner",
  "cross",
  "natural",
  "database",
  "drop",
  "grant",
];

// Note: `last` was removed from the list of built-in functions because it collides with the
// `last` table available in osquery
export const sqlBuiltinFunctions = [
  "avg",
  "count",
  "first",
  "max",
  "min",
  "sum",
  "ucase",
  "lcase",
  "mid",
  "len",
  "round",
  "rank",
  "now",
  "format",
  "coalesce",
  "ifnull",
  "isnull",
  "nvl",
];

export const sqlDataTypes = [
  "int",
  "numeric",
  "decimal",
  "date",
  "varchar",
  "char",
  "bigint",
  "float",
  "double",
  "bit",
  "binary",
  "text",
  "set",
  "timestamp",
  "money",
  "real",
  "number",
  "integer",
];

export default {
  checkPlatformCompatibility,
  checkTable,
  sqlKeyWords,
  sqlBuiltinFunctions,
  sqlDataTypes,
};
