// @ts-ignore
import { Parser } from "node-sql-parser";
import { intersection, isPlainObject, uniq } from "lodash";
import { osqueryTablesAvailable } from "utilities/osquery_tables";
import {
  MACADMINS_EXTENSION_TABLES,
  QUERYABLE_PLATFORMS,
  QueryablePlatform,
} from "interfaces/platform";
import { TableSchemaPlatform } from "interfaces/osquery_table";

const parser = new Parser();

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
  callback: (ast: IAstNode, parentKey: string) => void,
  parentKey = ""
) => {
  if (abstractSyntaxTree) {
    callback(abstractSyntaxTree, parentKey);

    Object.keys(abstractSyntaxTree).forEach((key) => {
      const childNode = abstractSyntaxTree[key];
      if (Array.isArray(childNode)) {
        childNode.forEach((grandchildNode) =>
          _visit(grandchildNode, callback, key)
        );
      } else if (childNode && _isNode(childNode)) {
        _visit(childNode, callback, key);
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
  includeVirtualTables = false
): string[] => {
  let results: string[] = [];

  // Tables defined via common table expression (WITH ... AS syntax) or as subselects
  // will be excluded from results by default.
  const virtualTables: string[] = [];

  // Tables defined via functions like `json_each` will always be excluded from results.
  const functionTables: string[] = [];

  const _callback = (node: IAstNode, parentKey: string) => {
    if (!node) {
      return;
    }

    // Common Table Expressions (CTEs) using "WITH ... AS" syntax.
    if (
      parentKey === "with" &&
      node.name &&
      (node.name as IAstNode).type === "default"
    ) {
      const withTable = node.name as IAstNode;
      if (typeof withTable.value === "string") {
        virtualTables.push(withTable.value);
      }
      return;
    }

    // Parse tables referenced by FROM or JOIN clauses.
    if (parentKey === "from" || parentKey === "left" || parentKey === "right") {
      // Subselects and JSON functions.
      if (node.expr) {
        // Check if the node is a function call.
        if ((node.expr as IAstNode).type === "function") {
          // Get the function name from node.expr.name.name[0].value
          // and push it to functionTables.
          const nodeExprName = (node.expr as IAstNode).name as IAstNode;
          const nodeExprNameArr = nodeExprName.name as IAstNode[];
          if (nodeExprNameArr.length > 0) {
            const functionName = nodeExprNameArr[0].value as string;
            if (functionName) {
              functionTables.push(functionName);
            }
          }
          return;
        }
        // Otherwise push it to the set of virtual tables.
        virtualTables.push(node.as as string);
        return;
      }

      // Plain ol' tables.
      if (node.table) {
        results.push(node.table as string);
      }
    }
  };

  try {
    const sqlTree = parser.astify(sqlString, { database: "sqlite" }) as unknown;
    _visit(sqlTree as IAstNode, _callback);

    // Remove virtual tables unless includeVirtualTables is true.
    if (virtualTables.length && !includeVirtualTables) {
      results = results.filter((r: string) => !virtualTables.includes(r));
    }

    // Always remove function tables.
    if (functionTables.length) {
      results = results.filter((r: string) => !functionTables.includes(r));
    }

    // Remove duplicates.
    results = uniq(results);

    return results;
  } catch (err) {
    // console.log(`sqlite-parser error: ${err}\n\n${sqlString}`);

    throw err;
  }
};

export const checkTable = (
  sqlString = "",
  includeVirtualTables = false
): { tables: string[] | null; error: Error | null } => {
  let sqlTables: string[] | undefined;
  try {
    sqlTables = parseSqlTables(sqlString, includeVirtualTables);
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
  includeVirtualTables = false
): { platforms: QueryablePlatform[] | null; error: Error | null } => {
  let sqlTables: string[] | undefined;
  try {
    // get tables from str
    sqlTables = parseSqlTables(sqlString, includeVirtualTables);
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
