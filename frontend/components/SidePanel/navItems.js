export default (admin) => {
  const adminNavItems = [
    {
      icon: 'kolidecon-admin',
      name: 'Admin',
      path: '/admin',
      subItems: [],
    },
  ];

  const userNavItems = [
    {
      icon: 'kolidecon-hosts',
      name: 'Hosts',
      path: '/admin/hosts',
      subItems: [
        { name: 'Add Hosts', path: '/' },
        { name: 'Manage Hosts', path: '/' },
      ],
    },
    {
      icon: 'kolidecon-query',
      name: 'Query',
      path: '/admin/queries',
      subItems: [
        { name: 'New Query', path: '/' },
        { name: 'Queries & Results', path: '/' },
      ],
    },
    {
      icon: 'kolidecon-packs',
      name: 'Packs',
      path: '/admin/packs',
      subItems: [],
    },
    {
      icon: 'kolidecon-alerts',
      name: 'Alerts',
      path: '/admin/alerts',
      subItems: [],
    },
    {
      icon: 'kolidecon-help',
      name: 'Help',
      path: '/admin/help',
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
