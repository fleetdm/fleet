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

const EmptySoftware = (message: string): JSX.Element => {
  const emptySoftware = (
    <div className={`${baseClass}__empty-software`}>
      <h1>
        No installed software{" "}
        {message === "vulnerable"
          ? "with detected vulnerabilities"
          : "detected"}
        .
      </h1>
      <p>
        Expecting to see{" "}
        {message === "vulnerable" && "detected vulnerabilities "}software? Check
        out the Fleet documentation on{" "}
        <a
          href="https://fleetdm.com/docs/deploying/configuration#software-inventory"
          target="_blank"
          rel="noopener noreferrer"
        >
          how to configure software inventory
        </a>
        .
      </p>
    </div>
  );

  switch (message) {
    case "modal":
      return (
        <div className={`${baseClass}__empty-software-modal`}>
          {emptySoftware}
        </div>
      );
    case "search":
      return (
        <div className={`${baseClass}__empty-software-modal`}>
          <div className={`${baseClass}__empty-software`}>
            <h1>No software matches the current search criteria.</h1>
            <p>
              Expecting to see software? Try again in a few seconds as the
              system catches up.
            </p>
          </div>
        </div>
      );
    default:
      return emptySoftware;
  }
};

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
              emptyComponent={EmptySoftware}
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
              emptyComponent={() => EmptySoftware("vulnerable")}
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
              emptyComponent={() =>
                EmptySoftware(
                  modalSoftwareSearchText === "" ? "modal" : "search"
                )
              }
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
