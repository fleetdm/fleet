export default (admin) => {
  const adminNavItems = [
    {
      icon: 'kolidecon-admin',
      name: 'Admin',
      path: {
        regex: /^\/admin/,
        location: '/admin/users',
      },
      subItems: [
        {
          name: 'Manage Users',
          path: {
            regex: /\/users/,
            location: '/admin/users',
          },
        },
        {
          name: 'App Settings',
          path: {
            regex: /\/settings/,
            location: '/admin/settings',
          },
        },
      ],
    },
  ];

  const userNavItems = [
    {
      icon: 'kolidecon-hosts',
      name: 'Hosts',
      path: {
        regex: /^\/hosts/,
        location: '/hosts/new',
      },
      subItems: [
        {
          name: 'Add Hosts',
          path: {
            regex: /\/new/,
            location: '/hosts/new',
          },
        },
        {
          name: 'Manage Hosts',
          path: {
            regex: /\/manage/,
            location: '/hosts/manage',
          },
        },
      ],
    },
    {
      defaultPathname: '/queries/new',
      icon: 'kolidecon-query',
      name: 'Query',
      path: {
        regex: /^\/queries/,
        location: '/queries/new',
      },
      subItems: [
        {
          name: 'New Query',
          path: {
            regex: /\/new/,
            location: '/queries/new',
          },
        },
        {
          name: 'Queries & Results',
          path: {
            regex: /\/results/,
            location: '/queries/results',
          },
        },
      ],
    },
    {
      defaultPathname: '/packs/all',
      icon: 'kolidecon-packs',
      name: 'Packs',
      path: {
        regex: /^\/packs/,
        location: '/packs/all',
      },
      subItems: [
        {
          name: 'All Packs',
          path: {
            regex: /\/all/,
            location: '/packs/all',
          },
        },
        {
          name: 'Pack Composer',
          path: {
            regex: /\/new/,
            location: '/packs/new',
          },
        },
      ],
    },
    {
      icon: 'kolidecon-help',
      name: 'Help',
      path: {
        regex: /^\/help/,
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
