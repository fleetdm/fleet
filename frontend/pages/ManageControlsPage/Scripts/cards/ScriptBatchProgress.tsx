import React, { useCallback, useState } from "react";
import { useQueryClient } from "react-query";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import PATHS from "router/paths";

import scriptsAPI, {
  IScriptBatchSummariesResponse,
  IScriptBatchSummaryResponse,
} from "services/entities/scripts";

import { isValidScriptBatchStatus, ScriptBatchStatus } from "interfaces/script";

import SectionHeader from "components/SectionHeader";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import PaginatedList from "components/PaginatedList";

import { IScriptsCommonProps } from "../ScriptsNavItems";
import { ScriptsLocation } from "../Scripts";

const baseClass = "script-batch-progress";

const STATUS_BY_INDEX: ScriptBatchStatus[] = [
  "started",
  "scheduled",
  "completed",
];

export type IScriptBatchProgressProps = IScriptsCommonProps & {
  location?: ScriptsLocation;
};

const ScriptBatchProgress = ({
  location,
  router,
  teamId,
}: IScriptBatchProgressProps) => {
  const handleTabChange = useCallback(
    (index: number) => {
      const newStatus = STATUS_BY_INDEX[index];
      // push to the URL
      const newParams = new URLSearchParams(location?.search);
      newParams.set("status", newStatus);
      const newQuery = newParams.toString();

      router.push(
        PATHS.CONTROLS_SCRIPTS_BATCH_PROGRESS.concat(
          newQuery ? `?${newQuery}` : ""
        )
      );
    },
    [location?.search, router]
  );
  const statusParam = location?.query.status;

  if (!isValidScriptBatchStatus(statusParam)) {
    handleTabChange(0); // Default to the first tab if the status is invalid
  }
  const selectedStatus = statusParam as ScriptBatchStatus;

  const queryClient = useQueryClient();
  const DEFAULT_PAGE_SIZE = 10;

  const fetchPage = useCallback((pageNumber: number) => {
    return queryClient.fetchQuery(
      [
        {
          team_id: teamId,
          status: selectedStatus,
          page: pageNumber,
          per_page: DEFAULT_PAGE_SIZE,
        },
      ],
      ({ queryKey }) => {
        return scriptsAPI
          .getRunScriptBatchSummaries(queryKey[0])
          .then((r) => r.batch_executions); // TODO - if `meta` field from response useful for PaginatedList, expand its functionality to handle generics other than just arrays of the expected object
      }
    );
  }, []);

  const renderPaginatedList = () => (
    <PaginatedList<IScriptBatchSummaryResponse>
      fetchPage={fetchPage}
      // onClickRow={() => {
      //   // TODO
      // }}
    />
  );

  return (
    <div className={baseClass}>
      <SectionHeader title="Batch progress" alignLeftHeaderVertically />
      <TabNav>
        <Tabs
          selectedIndex={STATUS_BY_INDEX.indexOf(selectedStatus)}
          onSelect={handleTabChange}
        >
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
