const path = require("path");

// Image/media imports resolve to a URL string at build time. Returning the
// filename rather than one shared stub keeps identity assertions meaningful —
// e.g. the FMA icon matcher tests, which check that a given software name
// resolves to a specific icon file.
module.exports = {
  process(src, filename) {
    return {
      code: `module.exports = ${JSON.stringify(path.basename(filename))};`,
    };
  },
  getCacheKey() {
    return "fleetFileTransform";
  },
};
