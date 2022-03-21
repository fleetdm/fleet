// @ts-ignore
import sqliteParser from "sqlite-parser";
import { intersection, isPlainObject } from "lodash";
// @ts-ignore
import { osqueryTables } from "utilities/osquery_tables";
import {
  IOsqueryPlatform,
  EXTENSION_TABLES,
  SUPPORTED_PLATFORMS,
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
  platforms: IOsqueryPlatform[];
}

export type IParserResult = Error | IOsqueryPlatform | string;

type IPlatformDictionay = Record<string, IOsqueryPlatform[]>;

const platformsByTableDictionary: IPlatformDictionay = (osqueryTables as IOsqueryTable[]).reduce(
  (dictionary: IPlatformDictionay, osqueryTable) => {
    dictionary[osqueryTable.name] = osqueryTable.platforms;
    return dictionary;
  },
  {}
);

Object.entries(EXTENSION_TABLES).forEach(([tableName, platforms]) => {
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

export const filterCompatiblePlatforms = (
  sqlTables: string[]
): IOsqueryPlatform[] => {
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

export const parseSqlTables = (
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
    // console.log(`Invalid query syntax: ${err.message}\n\n${sqlString}`);

    throw err; // TODO
  }
};

export const checkPlatformCompatibility = (
  sqlString: string,
  includeCteTables = false
): IOsqueryPlatform[] => {
  let sqlTables: string[] | undefined;
  try {
    sqlTables = parseSqlTables(sqlString, includeCteTables);
  } catch (err) {
    throw err;
  }

  if (sqlTables === undefined) {
    throw new Error(
      "Unexpected error checking platform compatibility: sqlTables are undefined"
    );
  }

  return filterCompatiblePlatforms(sqlTables);
};

export default checkPlatformCompatibility;
