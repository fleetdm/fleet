import { find } from 'lodash';

export const activeTabFromPathname = (navItems, pathname) => {
  return find(navItems, (item) => {
    return item.path.test(pathname);
  });
};

export const activeSubTabFromPathname = (activeTab, pathname) => {
  if (!activeTab) return undefined;

  const { subItems } = activeTab;

  if (!subItems.length) return undefined;

  return find(subItems, (subItem) => {
    return subItem.path.test(pathname);
  });
};
