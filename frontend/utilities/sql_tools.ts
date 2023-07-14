// @ts-ignore
import sqliteParser from "sqlite-parser";
import { intersection, isPlainObject } from "lodash";
import { osqueryTables } from "utilities/osquery_tables";
import {
  OsqueryPlatform,
  MACADMINS_EXTENSION_TABLES,
  SUPPORTED_PLATFORMS,
  SupportedPlatform,
} from "interfaces/platform";

type IAstNode = Record<string | number | symbol, unknown>;

interface ISqlTableNode {
  name: string;
}

interface ISqlCteNode {
  target: {
    name: string;
  };
}

// TODO: Research if there are any preexisting types for osquery schema
// TODO: Is it ever possible that osquery_tables.json would be missing name or platforms?
interface IOsqueryTable {
  name: string;
  platforms: OsqueryPlatform[];
}

type IPlatformDictionay = Record<string, OsqueryPlatform[]>;

const platformsByTableDictionary: IPlatformDictionay = (osqueryTables as IOsqueryTable[]).reduce(
  (dictionary: IPlatformDictionay, osqueryTable) => {
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
): SupportedPlatform[] => {
  if (!sqlTables.length) {
    return [...SUPPORTED_PLATFORMS]; // if a query has no tables but is still syntatically valid sql, it is treated as compatible with all platforms
  }

  const compatiblePlatforms = intersection(
    ...sqlTables.map(
      (tableName: string) => platformsByTableDictionary[tableName]
    )
  );

  return SUPPORTED_PLATFORMS.filter((p) => compatiblePlatforms.includes(p));
};

const parseSqlTables = (
  sqlString: string,
  includeCteTables = false
): string[] => {
  let results: string[] = [];

  // Tables defined via common table expression will be excluded from results by default
  const cteTables: string[] = [];

  const _callback = (node: IAstNode) => {
    if (node) {
      if (
        (node.variant === "common" || node.variant === "recursive") &&
        node.format === "table" &&
        node.type === "expression"
      ) {
        const targetName = ((node as unknown) as ISqlCteNode).target.name;
        cteTables.push(targetName);
      } else if (node.variant === "table") {
        const tableName = ((node as unknown) as ISqlTableNode).name;
        results.push(tableName);
      }
    }
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

const checkPlatformCompatibility = (
  sqlString: string,
  includeCteTables = false
): { platforms: SupportedPlatform[] | null; error: Error | null } => {
  let sqlTables: string[] | undefined;
  try {
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
    const platforms = filterCompatiblePlatforms(sqlTables);
    return { platforms, error: null };
  } catch (err) {
    return { platforms: null, error: new Error(`${err}`) };
  }
};

export default checkPlatformCompatibility;
