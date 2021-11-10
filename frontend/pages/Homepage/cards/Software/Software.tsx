import React, { useState } from "react";
import { useQuery } from "react-query";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";

import softwareAPI from "services/entities/software";
import { ISoftware } from "interfaces/software";

import Modal from "components/Modal";
import TabsWrapper from "components/TabsWrapper";
import TableContainer from "components/TableContainer";

import { generateTableHeaders } from "./SoftwareTableConfig";

interface ITableQueryProps {
  pageIndex: number;
  pageSize: number;
  searchQuery: string;
  sortHeader: string;
  sortDirection: string;
}

interface ISoftwareCardProps {
  isModalOpen: boolean;
  setIsSoftwareModalOpen: (isOpen: boolean) => void;
}

const PAGE_SIZE = 8;
const baseClass = "home-software";

const Software = ({
  isModalOpen,
  setIsSoftwareModalOpen,
}: ISoftwareCardProps): JSX.Element => {
  const [softwarePageIndex, setSoftwarePageIndex] = useState<number>(0);
  const [softwareSearchText, setSoftwareSearchText] = useState<string>("");
  const [vSoftwarePageIndex, setvSoftwarePageIndex] = useState<number>(0);
  const [vSoftwareSearchText, setvSoftwareSearchText] = useState<string>("");
  const [navTabIndex, setNavTabIndex] = useState<number>(0);

  const { data: software, isLoading: isLoadingSoftware } = useQuery<
    ISoftware[],
    Error
  >(["software", softwarePageIndex, softwareSearchText], () =>
    softwareAPI.load({
      page: softwarePageIndex,
      perPage: PAGE_SIZE,
      query: softwareSearchText,
    })
  );

  const {
    data: vulnerableSoftware,
    isLoading: isLoadingVulnerableSoftware,
  } = useQuery<ISoftware[], Error>(
    ["vSoftware", vSoftwarePageIndex, vSoftwareSearchText],
    () =>
      softwareAPI.load({
        page: softwarePageIndex,
        perPage: PAGE_SIZE,
        query: softwareSearchText,
      }),
    {
      select: (data: ISoftware[]) => data.filter((s) => s.vulnerabilities)
    }
  );

  // NOTE: this is called once on the initial rendering. The initial render of
  // the TableContainer child component will call this handler.
  const onAllSoftwareQueryChange = async ({
    pageIndex,
    searchQuery,
  }: ITableQueryProps) => {
    if (pageIndex === softwarePageIndex) {
      return false;
    }

    setSoftwarePageIndex(pageIndex);
    setSoftwareSearchText(searchQuery);
  };

  const onVulnerableSoftwareQueryChange = async ({
    pageIndex,
    searchQuery,
  }: ITableQueryProps) => {
    if (pageIndex === softwarePageIndex) {
      return false;
    }

    setvSoftwarePageIndex(pageIndex);
    setvSoftwareSearchText(searchQuery);
  };

  const tableHeaders = generateTableHeaders();
  // const vulnerableSoftware = software?.filter((s) => s.vulnerabilities);
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
              defaultSortHeader={"name"}
              defaultSortDirection={"asc"}
              hideActionButton
              resultsTitle={"software"}
              emptyComponent={() => <span>No software</span>}
              showMarkAllPages={false}
              isAllPagesSelected={false}
              disableCount
              disableActionButton
              pageSize={PAGE_SIZE}
              onQueryChange={onAllSoftwareQueryChange}
            />
          </TabPanel>
          <TabPanel>
            <TableContainer
              columns={tableHeaders}
              data={vulnerableSoftware || []}
              isLoading={isLoadingVulnerableSoftware}
              defaultSortHeader={"name"}
              defaultSortDirection={"asc"}
              hideActionButton
              resultsTitle={"software"}
              emptyComponent={() => <span>No vulnerable software</span>}
              showMarkAllPages={false}
              isAllPagesSelected={false}
              disableCount
              disableActionButton
              pageSize={PAGE_SIZE}
              onQueryChange={onVulnerableSoftwareQueryChange}
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
              defaultSortHeader={"name"}
              defaultSortDirection={"asc"}
              hideActionButton
              resultsTitle={"software items"}
              emptyComponent={() => <span>No vulnerable software</span>}
              showMarkAllPages={false}
              isAllPagesSelected={false}
              searchable
              disableActionButton
              pageSize={PAGE_SIZE}
              onQueryChange={onAllSoftwareQueryChange}
            />
          </>
        </Modal>
      )}
    </div>
  );
};

export default Software;
