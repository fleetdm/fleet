import React, { useState } from "react";
import { useQuery } from "react-query";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";

import softwareAPI from "services/entities/software";
import { ISoftware } from "interfaces/software";

import Modal from "components/modals/Modal";
import TabsWrapper from "components/TabsWrapper";
import TableContainer from "components/TableContainer";

import { generateTableHeaders } from "./SoftwareTableConfig";

interface ISoftwareCardProps {
  isModalOpen: boolean;
  setIsSoftwareModalOpen: (isOpen: boolean) => void;
}

const baseClass = "home-software";

const Software = ({
  isModalOpen,
  setIsSoftwareModalOpen,
}: ISoftwareCardProps): JSX.Element => {
  const [softwarePage, setSoftwarePage] = useState<number>(0);
  const [navTabIndex, setNavTabIndex] = useState<number>(0);

  const { data: software, isLoading: isLoadingSoftware } = useQuery<
    ISoftware[],
    Error
  >(["software", softwarePage], () => softwareAPI.load({}));

  const tableHeaders = generateTableHeaders();
  const vulnerableSoftware = software?.filter((s) => s.vulnerabilities);
  return (
    <div className={baseClass}>
      <TabsWrapper>
        <Tabs selectedIndex={navTabIndex} onSelect={(i) => setNavTabIndex(i)}>
          <TabList>
            <Tab>All</Tab>
            <Tab>Vulnerable</Tab>
          </TabList>
          <TabPanel>
            <TableContainer
              columns={tableHeaders}
              data={software || []}
              isLoading={isLoadingSoftware}
              defaultSortHeader={"hosts_count"}
              defaultSortDirection={"desc"}
              hideActionButton
              resultsTitle={"software"}
              emptyComponent={() => <span>No software</span>}
              showMarkAllPages={false}
              isAllPagesSelected={false}
              disableCount
              disableActionButton
            />
          </TabPanel>
          <TabPanel>
            <TableContainer
              columns={tableHeaders}
              data={vulnerableSoftware || []}
              isLoading={isLoadingSoftware}
              defaultSortHeader={"hosts_count"}
              defaultSortDirection={"desc"}
              hideActionButton
              resultsTitle={"software"}
              emptyComponent={() => <span>No vulnerable software</span>}
              showMarkAllPages={false}
              isAllPagesSelected={false}
              disableCount
              disableActionButton
            />
          </TabPanel>
        </Tabs>
      </TabsWrapper>
      {isModalOpen && (
        <Modal
          title="Software"
          onExit={() => setIsSoftwareModalOpen(false)}
          className={`${baseClass}__software-modal`}
        >
          <>
            <p>
              Search for a specific software version to find the hosts that have
              it installed.
            </p>
            <TableContainer
              columns={tableHeaders}
              data={software || []}
              isLoading={isLoadingSoftware}
              defaultSortHeader={"hosts_count"}
              defaultSortDirection={"desc"}
              hideActionButton
              resultsTitle={"software items"}
              emptyComponent={() => <span>No vulnerable software</span>}
              showMarkAllPages={false}
              isAllPagesSelected={false}
              searchable
              disableActionButton
            />
          </>
        </Modal>
      )}
    </div>
  );
};

export default Software;
