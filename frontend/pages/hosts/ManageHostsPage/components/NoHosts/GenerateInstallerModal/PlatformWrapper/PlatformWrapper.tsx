import React from "react";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import { push } from "react-router-redux";
import { useDispatch, useSelector } from "react-redux";
import { IConfig } from "interfaces/config";
import permissionUtils from "utilities/permissions";

interface IPlatformSubNav {
  name: string;
}

interface IRootState {
  app: {
    config: IConfig;
  };
}

const platformSubNav: IPlatformSubNav[] = [
  {
    name: "macOS",
  },
  {
    name: "Windows",
  },
  {
    name: "Linux (RPM)",
  },
  {
    name: "Linux (DEB)",
  },
];

interface IPlatformWrapperProp {
  // children: JSX.Element;
  // location: {
  //   pathname: string;
  // };
}

// const getTabIndex = (path: string): number => {
//   return platformSubNav.findIndex((navItem) => {
//     return navItem.pathname.includes(path);
//   });
// };

const baseClass = "platform-wrapper";

const PlatformWrapper = (props: IPlatformWrapperProp): JSX.Element => {
  const dispatch = useDispatch();

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__nav-header`}>
        <Tabs
        // selectedIndex={getTabIndex(pathname)}
        // selectedIndex={0}
        // onSelect={(i) => navigateToNav(i)}
        >
          <TabList>
            {platformSubNav.map((navItem) => {
              // Bolding text when the tab is active causes a layout shift
              // so we add a hidden pseudo element with the same text string
              return (
                <Tab key={navItem.name} data-text={navItem.name}>
                  {navItem.name}
                </Tab>
              );
            })}
          </TabList>
          <TabPanel>MacOs</TabPanel>
          <TabPanel>Windows</TabPanel>
          <TabPanel>Linux 1</TabPanel>
          <TabPanel>Linux 2</TabPanel>
        </Tabs>
      </div>
    </div>
  );
};

export default PlatformWrapper;
