// Image/media imports resolve to a URL string at build time. Returning the
// filename rather than one shared stub keeps identity assertions meaningful —
// e.g. the FMA icon matcher tests, which check that a given software name
// resolves to a specific icon file.
//
// Jest loads a transformer with require(), so this file stays CommonJS. The
// basename is taken with a regex rather than node:path to keep it importless.
module.exports = {
  process(src, filename) {
    const basename = filename.replace(/^.*[\\/]/, "");
    return {
      code: `module.exports = ${JSON.stringify(basename)};`,
    };
  },
  getCacheKey() {
    return "fleetFileTransform";
  },
};
