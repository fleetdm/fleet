/**
 * Announcement banner
 * (sails.config.announcementBanner)
 *
 * The blue glass banner shown just below the header on fleetdm.com.
 * This is the only place to change its text or link, or to turn it off.
 * (Rendered in views/layouts/layout.ejs; styled in assets/styles/layout.less.)
 */

module.exports.announcementBanner = {

  // Set to false to hide the banner everywhere.
  enabled: true,

  text: 'Join the Fleet Virtual Summit on November 10',
  // Optional shorter text for phones, so the label leaves room for the glass around it.
  textForSmallScreens: 'Virtual Summit · Nov 10',
  href: '/announcements/fleet-virtual-summit-2026',

  // Shown instead to visitors in Europe (based on Cloudflare's cf-ipcountry header).
  // Set to null to show the default banner to everyone.
  // > To preview locally, add ?simulateCountry=DE to the URL.
  forVisitorsInEurope: {
    text: 'Keep your device data in Europe',
    href: '/europe',
  },

  // Pages that never show the banner. A trailing `/*` also matches every page below that path.
  // Every other page shows it, except pages that hide the header links and the page the banner links to.
  hiddenOnPages: [
    '/pricing',
    '/contact',
    '/workshops',
    '/handbook/*',
    '/legal/*',
    '/better',
    '/download',
    '/admin/*',
    '/connect-vanta',
    '/vanta-authorization',
    '/query-generator',
    '/configuration-generator',
  ],

};
