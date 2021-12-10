import React, { useState, useEffect } from "react";
import { useQuery } from "react-query";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";

import softwareAPI from "services/entities/software";
import { ISoftware } from "interfaces/software"; // @ts-ignore
import debounce from "utilities/debounce";

import Modal from "components/Modal";
import TabsWrapper from "components/TabsWrapper";
import TableContainer from "components/TableContainer"; // @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";

import {
  generateTableHeaders,
  generateModalSoftwareTableHeaders,
} from "./SoftwareTableConfig";

interface ITableQueryProps {
  pageIndex: number;
  pageSize: number;
  searchQuery: string;
  sortHeader: string;
  sortDirection: string;
}

interface ISoftwareCardProps {
  currentTeamId?: number;
  isModalOpen: boolean;
  setIsSoftwareModalOpen: (isOpen: boolean) => void;
}

const VULNERABLE_OPTIONS = [
  {
    disabled: false,
    label: "All software",
    value: false,
    helpText: "All sofware installed on your hosts.",
  },
  {
    disabled: false,
    label: "Vulnerable software",
    value: true,
    helpText:
      "All software installed on your hosts with detected vulnerabilities.",
  },
];

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
  currentTeamId,
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
  const [
    isModalSoftwareVulnerable,
    setIsModalSoftwareVulnerable,
  ] = useState<boolean>(false);
  const [modalSoftwareState, setModalSoftwareState] = useState<ISoftware[]>([]);
  const [navTabIndex, setNavTabIndex] = useState<number>(0);
  const [isLoadingSoftware, setIsLoadingSoftware] = useState<boolean>(true);
  const [
    isLoadingVulnerableSoftware,
    setIsLoadingVulnerableSoftware,
  ] = useState<boolean>(false);
  const [isLoadingModalSoftware, setIsLoadingModalSoftware] = useState<boolean>(
    false
  );

  const { data: software } = useQuery<ISoftware[], Error>(
    ["software", softwarePageIndex, currentTeamId],
    () => {
      setIsLoadingSoftware(true);
      return softwareAPI.load({
        page: softwarePageIndex,
        perPage: PAGE_SIZE,
        orderKey: "name,id",
        orderDir: "asc",
        teamId: currentTeamId && currentTeamId,
      });
    },
    {
      enabled: navTabIndex === 0,
      // If keepPreviousData is enabled,
      // useQuery no longer returns isLoading when making new calls after load
      // So we manage our own load states
      keepPreviousData: true,
      onSuccess: () => {
        setIsLoadingSoftware(false);
      },
    }
  );

  const { data: vulnerableSoftware } = useQuery<ISoftware[], Error>(
    ["vSoftware", vSoftwarePageIndex, currentTeamId],
    () => {
      setIsLoadingVulnerableSoftware(true);
      return softwareAPI.load({
        page: vSoftwarePageIndex,
        perPage: PAGE_SIZE,
        orderKey: "name,id",
        orderDir: "asc",
        vulnerable: true,
        teamId: currentTeamId && currentTeamId,
      });
    },
    {
      enabled: navTabIndex === 1,
      refetchOnWindowFocus: false,
      keepPreviousData: true,
      onSuccess: () => {
        setIsLoadingVulnerableSoftware(false);
      },
    }
  );

  const { data: modalSoftware } = useQuery<ISoftware[], Error>(
    [
      "modalSoftware",
      modalSoftwarePageIndex,
      modalSoftwareSearchText,
      isModalSoftwareVulnerable,
      currentTeamId,
    ],
    () => {
      setIsLoadingModalSoftware(true);
      return softwareAPI.load({
        page: modalSoftwarePageIndex,
        query: modalSoftwareSearchText,
        orderKey: "id",
        orderDir: "desc",
        vulnerable: isModalSoftwareVulnerable,
        teamId: currentTeamId && currentTeamId,
      });
    },
    {
      enabled: isModalOpen,
      refetchOnWindowFocus: false,
      keepPreviousData: true,
      onSuccess: () => {
        setIsLoadingModalSoftware(false);
      },
    }
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

  const onModalSoftwareQueryChange = debounce(
    async ({ pageIndex, searchQuery }: ITableQueryProps) => {
      setModalSoftwareSearchText(searchQuery);

      if (pageIndex !== modalSoftwarePageIndex) {
        setModalSoftwarePageIndex(pageIndex);
      }
    },
    { leading: false, trailing: true }
  );

  useEffect(() => {
    setModalSoftwareState(() => {
      return (
        modalSoftware?.filter((softwareItem) => {
          return softwareItem.name
            .toLowerCase()
            .includes(modalSoftwareSearchText.toLowerCase());
        }) || []
      );
    });
  }, [modalSoftware, modalSoftwareSearchText]);

  const renderStatusDropdown = () => {
    return (
      <Dropdown
        value={isModalSoftwareVulnerable}
        className={`${baseClass}__status_dropdown`}
        options={VULNERABLE_OPTIONS}
        searchable={false}
        onChange={(value: boolean) => setIsModalSoftwareVulnerable(value)}
      />
    );
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
              defaultSortHeader={"name"}
              defaultSortDirection={"asc"}
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
              defaultSortHeader={"name"}
              defaultSortDirection={"asc"}
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
              columns={generateModalSoftwareTableHeaders()}
              data={modalSoftwareState}
              isLoading={isLoadingModalSoftware}
              defaultSortHeader={"name"}
              defaultSortDirection={"asc"}
              hideActionButton
              filteredCount={modalSoftwareState.length}
              resultsTitle={"software items"}
              emptyComponent={() =>
                EmptySoftware(
                  modalSoftwareSearchText === "" ? "modal" : "search"
                )
              }
              showMarkAllPages={false}
              isAllPagesSelected={false}
              searchable
              disableActionButton
              pageSize={MODAL_PAGE_SIZE}
              onQueryChange={onModalSoftwareQueryChange}
              customControl={renderStatusDropdown}
              isClientSidePagination
              isClientSideSearch
            />
          </>
        </Modal>
      )}
    </div>
  );
};

export default Software;
