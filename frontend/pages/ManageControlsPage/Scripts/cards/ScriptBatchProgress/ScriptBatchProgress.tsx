import React, { useCallback, useEffect, useRef, useState } from "react";
import { useQuery } from "react-query";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import PATHS from "router/paths";

import scriptsAPI, {
  IScriptBatchSummaryV2,
  IScriptBatchSummariesResponse,
} from "services/entities/scripts";

import { isValidScriptBatchStatus, ScriptBatchStatus } from "interfaces/script";

import { COLORS } from "styles/var/colors";

import Spinner from "components/Spinner";
import ProgressBar from "components/ProgressBar";
import SectionHeader from "components/SectionHeader";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import PaginatedList, { IPaginatedListHandle } from "components/PaginatedList";
import Icon from "components/Icon/Icon";

import { IScriptsCommonProps } from "../../ScriptsNavItems";
import getWhen from "../../helpers";

const baseClass = "script-batch-progress";

const STATUS_BY_INDEX: ScriptBatchStatus[] = [
  "started",
  "scheduled",
  "finished",
];

export const EMPTY_STATE_DETAILS: Record<ScriptBatchStatus, string> = {
  started: "When a script is run on multiple hosts, progress will appear here.",
  scheduled:
    "When a script is scheduled to run in the future, it will appear here.",
  finished:
    "When a batch script is completed or canceled, historical results will appear here.",
};

const getEmptyState = (status: ScriptBatchStatus) => {
  return (
    <div className={`${baseClass}__empty`}>
      <b>No batch scripts {status} for this team</b>
      <p>{EMPTY_STATE_DETAILS[status]}</p>
    </div>
  );
};

export type IScriptBatchProgressProps = IScriptsCommonProps;

const ScriptBatchProgress = ({
  location,
  router,
  teamId,
}: IScriptBatchProgressProps) => {
  const [pageNumber, setPageNumber] = useState(0);

  const paginatedListRef = useRef<IPaginatedListHandle<IScriptBatchSummaryV2>>(
    null
  );

  const statusParam = location?.query.status;

  const selectedStatus = statusParam as ScriptBatchStatus;

  const DEFAULT_PAGE_SIZE = 10;

  const queryKey = {
    team_id: teamId,
    status: selectedStatus,
    page: pageNumber,
    per_page: DEFAULT_PAGE_SIZE,
  };

  const { data, isFetching: updating } = useQuery<
    IScriptBatchSummariesResponse,
    Error,
    IScriptBatchSummariesResponse
  >([queryKey], () => scriptsAPI.getRunScriptBatchSummaries(queryKey), {
    keepPreviousData: true,
  });

  const handleTabChange = useCallback(
    (index: number) => {
      const newStatus = STATUS_BY_INDEX[index];

      const newParams = new URLSearchParams(location?.search);
      newParams.set("status", newStatus);
      const newQuery = newParams.toString();

      router.push(
        PATHS.CONTROLS_SCRIPTS_BATCH_PROGRESS.concat(
          newQuery ? `?${newQuery}` : ""
        )
      );
      setPageNumber(0);
    },
    [location?.search, router]
  );

  const onClickRow = (r: IScriptBatchSummaryV2) => {
    // explicitly including the status param here avoids triggering the script details page's effect
    // which would add it automatically, muddying browser history and preventing smooth forward/back navigation
    router.push(
      PATHS.CONTROLS_SCRIPTS_BATCH_DETAILS(r.batch_execution_id).concat(
        "?status=ran"
      )
    );
    return r;
  };

  const renderRow = (summary: IScriptBatchSummaryV2) => {
    const {
      script_name,
      targeted_host_count,
      ran_host_count,
      errored_host_count,
    } = summary;
    const when = getWhen(summary);
    return (
      <>
        <div className={`${baseClass}__row-left`}>
          <b>{script_name}</b>
          <div className={`${baseClass}__row-when`}>{when}</div>
        </div>
        {summary.status !== "scheduled" && (
          <div className={`${baseClass}__row-right`}>
            <div>
              {ran_host_count + errored_host_count} / {targeted_host_count}{" "}
              hosts
            </div>
            <ProgressBar
              sections={[
                {
                  // results
                  color: COLORS["status-success"],
                  portion: ran_host_count / targeted_host_count,
                },
                {
                  // errors
                  color: COLORS["status-error"],
                  portion: errored_host_count / targeted_host_count,
                },
              ]}
            />
            <div className={`${baseClass}__row-errors`}>
              <Icon
                name="error-outline"
                color="ui-fleet-black-50"
                size="small"
              />{" "}
              <div>{errored_host_count}</div>
            </div>
          </div>
        )}
      </>
    );
  };

  // Reset to first tab if status is invalid.
  useEffect(() => {
    if (!isValidScriptBatchStatus(statusParam)) {
      handleTabChange(0);
    }
  }, [statusParam, handleTabChange]);

  const renderTabContent = (status: ScriptBatchStatus) => {
    // If we're switching to a new tab, show the loading spinner
    // while we get the first page and # of results.
    if (updating) {
      return (
        <div className={`${baseClass}__loading`}>
          <Spinner />
        </div>
      );
    }

    const count = data?.count || 0;
    const rows = data?.batch_executions || [];

    if (count === 0) {
      return getEmptyState(status);
    }

    return (
      <div className={`${baseClass}__tab-content`}>
        {!updating && count && (
          <div className={`${baseClass}__status-count`}>
            {count} batch script{count > 1 ? "s" : ""}
          </div>
        )}
        <PaginatedList<IScriptBatchSummaryV2>
          ref={paginatedListRef}
          count={count}
          data={rows}
          pageSize={DEFAULT_PAGE_SIZE}
          currentPage={pageNumber}
          onPageIndexChange={setPageNumber}
          isLoading={updating}
          onClickRow={onClickRow}
          renderItemRow={renderRow}
          useCheckBoxes={false}
        />
      </div>
    );
  };

  return (
    <>
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
                <TabText>Finished</TabText>
              </Tab>
            </TabList>
            <TabPanel>{renderTabContent(STATUS_BY_INDEX[0])}</TabPanel>
            <TabPanel>{renderTabContent(STATUS_BY_INDEX[1])}</TabPanel>
            <TabPanel>{renderTabContent(STATUS_BY_INDEX[2])}</TabPanel>
          </Tabs>
        </TabNav>
      </div>
    </>
  );
};

export default ScriptBatchProgress;
