import React, { useCallback, useState } from "react";
import { useQueryClient } from "react-query";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import PATHS from "router/paths";

import scriptsAPI, {
  IScriptBatchSummaryResponseV1,
  IScriptBatchSummaryV2,
} from "services/entities/scripts";

import { isValidScriptBatchStatus, ScriptBatchStatus } from "interfaces/script";

import SectionHeader from "components/SectionHeader";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import PaginatedList from "components/PaginatedList";
import Spinner from "components/Spinner";

import { IScriptsCommonProps } from "../ScriptsNavItems";
import { ScriptsLocation } from "../Scripts";

const baseClass = "script-batch-progress";

const STATUS_BY_INDEX: ScriptBatchStatus[] = [
  "started",
  "scheduled",
  "completed",
];

const EMPTY_STATE_DETAILS: Record<ScriptBatchStatus, string> = {
  started: "When a script is run on multiple hosts, progress will appear here.",
  scheduled:
    "When a script is scheduled to run in the future, it will appear here.",
  completed:
    "When a batch script is completed or cancelled, historical results will appear here.",
};

export type IScriptBatchProgressProps = IScriptsCommonProps & {
  location?: ScriptsLocation;
};

const ScriptBatchProgress = ({
  location,
  router,
  teamId,
}: IScriptBatchProgressProps) => {
  const [batchCount, setBatchCount] = useState<number | undefined>(undefined);
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

  const fetchPage = useCallback(
    (pageNumber: number) => {
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
            .then((r) => {
              setBatchCount(r.count);
              return r.batch_executions;
            });
        }
      );
    },
    [queryClient, selectedStatus, teamId]
  );

  const onClickRow = (r: IScriptBatchSummaryV2) => {
    // TODO - summary modal for now
    return r;
  };

  const getWhen = (
    summary: IScriptBatchSummaryV2,
    status: ScriptBatchStatus
  ) => {
    return "TODO";
  };

  const renderRow = (summary: IScriptBatchSummaryV2) => {
    const when = getWhen(summary, selectedStatus);
    return (
      <>
        <div className={`${baseClass}__row-left`}>
          <b>{summary.script_name}</b>
          <div className={`${baseClass}__row-when`}>{when}</div>
        </div>
        <div className={`${baseClass}__row-right`}>TODO - progress bar</div>
      </>
    );
  };

  const getEmptyState = (status: ScriptBatchStatus) => {
    return (
      <>
        <b>No batch scripts {status} for this team</b>
        <p>{EMPTY_STATE_DETAILS[status]}</p>
      </>
    );
  };

  const heading = batchCount ? (
    <p>
      <b>
        {batchCount} batch script{batchCount > 1 && "s"}
      </b>
    </p>
  ) : undefined;

  const renderTabContent = (status: ScriptBatchStatus) => {
    if (batchCount === 0) {
      return getEmptyState(status);
    }
    return (
      <>
        <PaginatedList<IScriptBatchSummaryV2>
          heading={heading}
          count={batchCount}
          fetchPage={fetchPage}
          onClickRow={onClickRow}
          renderItemRow={renderRow}
          useCheckBoxes={false}
        />
      </>
    );
  };

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
          <TabPanel>{renderTabContent(STATUS_BY_INDEX[0])}</TabPanel>
          <TabPanel>{renderTabContent(STATUS_BY_INDEX[1])}</TabPanel>
          <TabPanel>{renderTabContent(STATUS_BY_INDEX[2])}</TabPanel>
        </Tabs>
      </TabNav>
    </div>
  );
};

export default ScriptBatchProgress;
