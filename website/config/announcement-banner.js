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

  // The banner shown to everyone (in the shape `{ text, textForSmallScreens, href }`).
  // Set to null to show no banner to visitors outside Europe.
  // > Optional `textForSmallScreens` is a shorter label for phones, so the text leaves room for the glass around it.
  forEveryone: null,

  // Shown instead to visitors in Europe (based on Cloudflare's cf-ipcountry header).
  // Set to null to show nothing to visitors in Europe (unless `forEveryone` is set).
  // > To preview locally, add ?simulateCountry=DE to the URL.
  forVisitorsInEurope: {
    text: 'Fleet is now ISO 27001 certified',
    textForSmallScreens: 'ISO 27001 certified',
    href: '/announcements/fleet-accelerates-european-expansion',
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
