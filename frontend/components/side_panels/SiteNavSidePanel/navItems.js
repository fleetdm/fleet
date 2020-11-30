import PATHS from 'router/paths';
import URL_PREFIX from 'router/url_prefix';

export default (admin) => {
  const adminNavItems = [
    {
      icon: 'admin',
      name: 'Admin',
      location: {
        regex: new RegExp(`^${URL_PREFIX}/admin/`),
        pathname: PATHS.ADMIN_USERS,
      },
      subItems: [
        {
          icon: 'admin',
          name: 'Manage Users',
          location: {
            regex: new RegExp(`^${PATHS.ADMIN_USERS}`),
            pathname: PATHS.ADMIN_USERS,
          },
        },
        {
          icon: 'user-settings',
          name: 'App Settings',
          location: {
            regex: new RegExp(`^${PATHS.ADMIN_SETTINGS}`),
            pathname: PATHS.ADMIN_SETTINGS,
          },
        },
        {
          // No such icon now. SiteNavSidePanel does not display
          // icons for subItems
          icon: 'osquery',
          name: 'Osquery Options',
          location: {
            regex: new RegExp(`^${PATHS.ADMIN_OSQUERY}`),
            pathname: PATHS.ADMIN_OSQUERY,
          },
        },
      ],
    },
  ];

  const userNavItems = [
    {
      icon: 'hosts',
      name: 'Hosts',
      location: {
        regex: new RegExp(`^${URL_PREFIX}/hosts/`),
        pathname: PATHS.MANAGE_HOSTS,
      },
      subItems: [],
    },
    {
      icon: 'query',
      name: 'Query',
      location: {
        regex: new RegExp(`^${URL_PREFIX}/queries/`),
        pathname: PATHS.MANAGE_QUERIES,
      },
      subItems: [],
    },
    {
      icon: 'packs',
      name: 'Packs',
      location: {
        regex: new RegExp(`^${URL_PREFIX}/packs/`),
        pathname: PATHS.MANAGE_PACKS,
      },
      subItems: [],
    },
    {
      icon: 'help',
      name: 'Help',
      location: {
        regex: /^\/help/,
        pathname: 'https://github.com/fleetdm/fleet/blob/master/docs/README.md',
      },
      subItems: [],
    },
  ];

  if (admin) {
    return [
      ...userNavItems,
      ...adminNavItems,
    ];
  }

  return userNavItems;
};
