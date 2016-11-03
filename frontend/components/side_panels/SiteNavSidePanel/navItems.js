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
          name: 'User Management',
          path: {
            regex: /\/users/,
            location: '/admin/users',
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
      icon: 'kolidecon-packs',
      name: 'Packs',
      path: {
        regex: /^\/packs/,
      },
      subItems: [],
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
