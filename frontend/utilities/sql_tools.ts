// @ts-ignore
import sqliteParser from "sqlite-parser";
import { intersection, isPlainObject } from "lodash";
// @ts-ignore
import { osqueryTables } from "utilities/osquery_tables";

type IAstNode = Record<string | number | symbol, unknown>;

interface ISqlTableNode {
  name: string;
}

interface ISqlCteNode {
  target: {
    name: string;
  };
}
export type IOsqueryPlatform = "darwin" | "windows" | "linux" | "freebsd";

// TODO: Research if there are any preexisting types for osquery schema
// TODO: Is it ever possible that osquery_tables.json would be missing name or platforms?
interface IOsqueryTable {
  name: string;
  platforms: IOsqueryPlatform[];
}

export const SUPPORTED_PLATFORMS = ["darwin", "windows", "linux"] as const;

export type IParserResult =
  | "invalid query syntax"
  | "no tables in query AST"
  | IOsqueryPlatform
  | string;

export type ICompatiblePlatform =
  | "all"
  | "none"
  | "invalid query syntax"
  | IOsqueryPlatform;

type IPlatformDictionay = Record<string, IOsqueryPlatform[]>;

const platformsByTableDictionary: IPlatformDictionay = (osqueryTables as IOsqueryTable[]).reduce(
  (dictionary: IPlatformDictionay, osqueryTable) => {
    dictionary[osqueryTable.name] = osqueryTable.platforms;
    return dictionary;
  },
  {}
);

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

export const filterPlatforms = (
  parserResults: IParserResult[]
): ICompatiblePlatform[] => {
  console.log(osqueryTables);
  if (parserResults[0] === "invalid query syntax") {
    return ["invalid query syntax"];
  }
  // if a query has no tables but is still syntatically valid sql, it is treated as compatible with all platforms
  if (parserResults[0] === "no tables in query AST") {
    return ["all"];
  }

  const compatiblePlatforms = intersection(
    ...parserResults?.map(
      (tableName: IParserResult) => platformsByTableDictionary[tableName]
    )
  );

  return compatiblePlatforms.length ? compatiblePlatforms : ["none"];
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

    return results.length ? results : ["no tables in query AST"];
  } catch (err) {
    // console.log(`Invalid query syntax: ${err.message}\n\n${sqlString}`);

    return ["invalid query syntax"];
  }
};

export const getCompatiblePlatforms = (
  sqlString: string,
  includeCteTables = false
): ICompatiblePlatform[] => {
  return filterPlatforms(parseSqlTables(sqlString, includeCteTables));
};

export default getCompatiblePlatforms;
