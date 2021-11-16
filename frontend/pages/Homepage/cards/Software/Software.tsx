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
const MODAL_PAGE_SIZE = 20;
const baseClass = "home-software";

const Software = ({
  isModalOpen,
  setIsSoftwareModalOpen,
}: ISoftwareCardProps): JSX.Element => {
  const [softwarePageIndex, setSoftwarePageIndex] = useState<number>(0);
  const [vSoftwarePageIndex, setVSoftwarePageIndex] = useState<number>(0);
  const [modalSoftwarePageIndex, setModalSoftwarePageIndex] = useState<number>(
    0
  );
  const [
    modalSoftwareSearchText,
    setModalSoftwareSearchText,
  ] = useState<string>("");
  const [navTabIndex, setNavTabIndex] = useState<number>(0);

  const { data: software, isLoading: isLoadingSoftware } = useQuery<
    ISoftware[],
    Error
  >(["software", softwarePageIndex], () =>
    softwareAPI.load({
      page: softwarePageIndex,
      perPage: PAGE_SIZE,
      orderKey: "host_count",
      orderDir: "desc",
    })
  );

  const {
    data: vulnerableSoftware,
    isLoading: isLoadingVulnerableSoftware,
  } = useQuery<ISoftware[], Error>(
    ["vSoftware", vSoftwarePageIndex],
    () =>
      softwareAPI.load({
        page: vSoftwarePageIndex,
        perPage: PAGE_SIZE,
        orderKey: "host_count",
        orderDir: "desc",
      }),
    {
      select: (data: ISoftware[]) => data.filter((s) => s.vulnerabilities),
    }
  );

  const { data: modalSoftware, isLoading: isLoadingModalSoftware } = useQuery<
    ISoftware[],
    Error
  >(["modalSoftware", modalSoftwarePageIndex, modalSoftwareSearchText], () =>
    softwareAPI.load({
      page: modalSoftwarePageIndex,
      perPage: MODAL_PAGE_SIZE,
      query: modalSoftwareSearchText,
      orderKey: "host_count",
      orderDir: "desc",
    })
  );

  // NOTE: this is called once on the initial rendering. The initial render of
  // the TableContainer child component will call this handler.
  const onAllSoftwareQueryChange = async ({ pageIndex }: ITableQueryProps) => {
    if (pageIndex !== softwarePageIndex) {
      setSoftwarePageIndex(pageIndex);
    }
  };

  const onVulnerableSoftwareQueryChange = async ({
    pageIndex,
  }: ITableQueryProps) => {
    if (pageIndex !== vSoftwarePageIndex) {
      setVSoftwarePageIndex(pageIndex);
    }
  };

  const onModalSoftwareQueryChange = async ({
    pageIndex,
    searchQuery,
  }: ITableQueryProps) => {
    setModalSoftwareSearchText(searchQuery);

    if (pageIndex !== modalSoftwarePageIndex) {
      setModalSoftwarePageIndex(pageIndex);
    }
  };

  const tableHeaders = generateTableHeaders();
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
              defaultSortHeader={"host_count"}
              defaultSortDirection={"desc"}
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
              defaultSortHeader={"host_count"}
              defaultSortDirection={"desc"}
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
              data={modalSoftware || []}
              isLoading={isLoadingModalSoftware}
              defaultSortHeader={"host_count"}
              defaultSortDirection={"desc"}
              hideActionButton
              resultsTitle={"software items"}
              emptyComponent={() => <span>No vulnerable software</span>}
              showMarkAllPages={false}
              isAllPagesSelected={false}
              searchable
              disableCount
              disableActionButton
              pageSize={MODAL_PAGE_SIZE}
              onQueryChange={onModalSoftwareQueryChange}
            />
          </>
        </Modal>
      )}
    </div>
  );
};

export default Software;
