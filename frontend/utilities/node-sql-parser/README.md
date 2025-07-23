# About

This is compiled from [a branch of the node-sql-parser library](https://github.com/sgress454/node-sql-parser/tree/sgress454/add-escape-to-sqlite-like) created to fix issue [#30109](https://github.com/fleetdm/fleet/issues/30109).

The new code has been [merged into the main branch of node-sql-parser](https://github.com/taozhi8833998/node-sql-parser/pull/2496) but has yet to be released.

Once a new release of node-sql-parser comes out with this code in it, we can revert back to using `node-sql-parser` as a dependency in Fleet.

# To compile

1. Check out https://github.com/taozhi8833998/node-sql-parser
2. `npm install`
3. `npm run build`
4. `cp output/prod/build/sqlite.* /path/to/fleet/frontend/utilities/node-sql-parser`