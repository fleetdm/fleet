# About

This is compiled from a [fork](https://github.com/taozhi8833998/node-sql-parser/compare/master...sgress454:node-sql-parser:5.3.10-plus) of the 5.3.10 release of [node-sql-parser library](https://github.com/sgress454/node-sql-parser/tree/sgress454/add-escape-to-sqlite-like) created to fix issue [#30109](https://github.com/fleetdm/fleet/issues/30109).

Once a new release of node-sql-parser comes out with this code in it, we can revert back to using `node-sql-parser` as a dependency in Fleet.

# To compile

1. Check out https://github.com/sgress454/node-sql-parser/tree/5.3.10-plus
2. `npm install`
3. `npm run build`
4. `cp output/prod/build/sqlite.* /path/to/fleet/frontend/utilities/node-sql-parser`