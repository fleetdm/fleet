import { find } from 'lodash';

export const activeTabFromPathname = (navItems, pathname) => {
  return find(navItems, (item) => {
    const { path: { regex } } = item;
    return regex.test(pathname);
  });
};

export const activeSubTabFromPathname = (activeTab, pathname) => {
  if (!activeTab) return undefined;

  const { subItems } = activeTab;

  if (!subItems.length) return undefined;

  return find(subItems, (subItem) => {
    const { path: { regex } } = subItem;
    return regex.test(pathname);
  }) || subItems[0];
};
