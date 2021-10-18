import React, { useState } from "react";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";

import { ISoftware } from "interfaces/software";

import Modal from "components/modals/Modal";
import TabsWrapper from "components/TabsWrapper";

interface ISoftwareCardProps {
  software: ISoftware[] | undefined;
  isModalOpen: boolean;
  setIsSoftwareModalOpen: (isOpen: boolean) => void;
}

const baseClass = "home-software";

const Software = ({
  software,
  isModalOpen,
  setIsSoftwareModalOpen,
}: ISoftwareCardProps): JSX.Element => {
  const [navTabIndex, setNavTabIndex] = useState(0);

  return (
    <div className={baseClass}>
      <TabsWrapper>
        <Tabs selectedIndex={navTabIndex} onSelect={(i) => setNavTabIndex(i)}>
          <TabList>
            <Tab>All</Tab>
            <Tab>Vulnerable</Tab>
          </TabList>
          <TabPanel>1</TabPanel>
          <TabPanel>2</TabPanel>
        </Tabs>
      </TabsWrapper>
      {isModalOpen && (
        <Modal
          title="Software"
          onExit={() => setIsSoftwareModalOpen(false)}
          className={`${baseClass}__software-modal`}
        >
          <div>3</div>
        </Modal>
      )}
    </div>
  );
};

export default Software;
