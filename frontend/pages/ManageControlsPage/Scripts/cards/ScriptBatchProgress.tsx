import React, { useCallback, useState } from "react";
import { useQueryClient } from "react-query";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import scriptsAPI, {
  IScriptBatchSummariesResponse,
} from "services/entities/scripts";

import { ScriptBatchStatus } from "interfaces/script";

import SectionHeader from "components/SectionHeader";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import PaginatedList from "components/PaginatedList";

import { IScriptsCommonProps } from "../ScriptsNavItems";

const baseClass = "script-batch-progress";

export type IScriptBatchProgressProps = IScriptsCommonProps;

const ScriptBatchProgress = ({ router, teamId }: IScriptBatchProgressProps) => {
  const [selectedStatus, setSelectedStatus] = useState<ScriptBatchStatus>(
    "started"
  ); // TODO - default to URL val

  const statusByIndex: ScriptBatchStatus[] = [
    "started",
    "scheduled",
    "completed",
  ];

  const handleTabChange = (index: number) => {
    // TODO - coordinate with URL here
    // TODO - coordinate stage string with below fetchPage
    setSelectedStatus(index);
  };

  // const { data: summaries, isLoading, error } = useQuery<IScriptBatchSummariesResponse,
  // AxiosError, >(() => {}, {});

  const queryClient = useQueryClient();
  const DEFAULT_PAGE_SIZE = 10;
  // const DEFAULT_SORT_COLUMN = "name";

  const fetchPage = useCallback((pageNumber: number) => {
    return queryClient.fetchQuery(
      [
        {
          team_id: teamId,
          status: "started" as ScriptBatchStatus, // TODO - make dynamic with tab nav
          page: pageNumber,
          per_page: DEFAULT_PAGE_SIZE,
        },
      ],
      ({ queryKey }) => {
        return scriptsAPI.getRunScriptBatchSummaries(queryKey[0]);
      }
    );
  }, []);

  const renderPaginatedList = () => (
    <PaginatedList
      fetchPage={fetchPage}
      onClickRow={() => {
        // TODO
      }}
    />
  );

  return (
    <div className={baseClass}>
      <SectionHeader title="Batch progress" alignLeftHeaderVertically />
      <TabNav>
        <Tabs selectedIndex={selectedStatus} onSelect={handleTabChange}>
          <TabList>
            <Tab>
              <TabText>Started</TabText>
            </Tab>
            <Tab>
              <TabText>Scheduled</TabText>
            </Tab>
            <Tab>
              <TabText>Completed</TabText>
            </Tab>
          </TabList>
          <TabPanel>{renderPaginatedList()}</TabPanel>
          <TabPanel>{renderPaginatedList()}</TabPanel>
          <TabPanel>{renderPaginatedList()}</TabPanel>
        </Tabs>
      </TabNav>
    </div>
  );
};

export default ScriptBatchProgress;
