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

  // Pages that show the banner. A trailing `/*` also matches every page below that path.
  // Anything not listed here (e.g. /pricing, /contact, /workshops, /login, /start, /handbook,
  // ad landing pages under /lp/) never shows it, nor does the page the banner links to.
  pages: [
    '/',
    '/docs/*',
    '/guides/*',
    '/tables/*',
    '/software-catalog/*',
    '/reports/*',
    '/policies/*',
    '/scripts/*',
    '/mdm-commands/*',
    '/vitals/*',
    '/os-settings',
    '/articles/*',
    '/announcements/*',
    '/releases/*',
    '/securing/*',
    '/engineering/*',
    '/podcasts/*',
    '/case-study/*',
    '/customers',
    '/partners',
    '/integrations',
    '/device-management',
    '/endpoint-governance',
    '/software-management',
    '/infrastructure-as-code',
    '/linux-management',
    '/arch-linux-management',
    '/security-and-control',
    '/ai',
    '/capex-savings',
  ],

};
