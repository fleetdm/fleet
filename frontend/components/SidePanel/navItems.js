export default (admin) => {
  const adminNavItems = [
    {
      defaultPathname: '/admin/users',
      icon: 'kolidecon-admin',
      name: 'Admin',
      path: /^\/admin/,
      subItems: [
        { name: 'User Management', path: /\/users/ },
      ],
    },
  ];

  const userNavItems = [
    {
      defaultPathname: '/hosts/new',
      icon: 'kolidecon-hosts',
      name: 'Hosts',
      path: /^\/hosts/,
      subItems: [
        { name: 'Add Hosts', path: /\/new/ },
        { name: 'Manage Hosts', path: /\/manage/ },
      ],
    },
    {
      defaultPathname: '/queries/new',
      icon: 'kolidecon-query',
      name: 'Query',
      path: /^\/queries/,
      subItems: [
        { name: 'New Query', path: /\/new/ },
        { name: 'Queries & Results', path: /\/results/ },
      ],
    },
    {
      icon: 'kolidecon-packs',
      name: 'Packs',
      path: /^\/packs/,
      subItems: [],
    },
    {
      icon: 'kolidecon-alerts',
      name: 'Alerts',
      path: /^\/alerts/,
      subItems: [],
    },
    {
      icon: 'kolidecon-help',
      name: 'Help',
      path: /^\/help/,
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
