/**
 * Configuration profile generator limits
 * (sails.config.profileGeneratorLimits)
 *
 * How many requests a Fleet server can make to the configuration profile generator in a UTC calendar month.
 * Fleet Free is limited per registration, and Fleet Premium per license key, shared by every Fleet server
 * that sends it.
 *
 * Kept in their own file so the generator's tests run on a pull request that changes them.
 * (.github/workflows/test-configuration-profile-generator.yml)
 */

module.exports.profileGeneratorLimits = {

  fleetFree: {
    profiles: 15,
    // Names and descriptions stop when profiles do, so this only matters to a caller that asks for one
    // without the other.
    namesAndDescriptions: 45,
  },

  fleetPremium: {
    profiles: 300,
    namesAndDescriptions: 900,
  },

};
