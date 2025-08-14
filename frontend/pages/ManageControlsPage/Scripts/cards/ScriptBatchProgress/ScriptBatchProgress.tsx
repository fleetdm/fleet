import React, { useCallback, useEffect, useState } from "react";
import { useQueryClient } from "react-query";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import PATHS from "router/paths";

import scriptsAPI, { IScriptBatchSummaryV2 } from "services/entities/scripts";

import { isValidScriptBatchStatus, ScriptBatchStatus } from "interfaces/script";

import { isDateTimePast } from "utilities/helpers";

import { COLORS } from "styles/var/colors";

import Spinner from "components/Spinner";
import ProgressBar from "components/ProgressBar";
import SectionHeader from "components/SectionHeader";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import PaginatedList from "components/PaginatedList";
import { HumanTimeDiffWithFleetLaunchCutoff } from "components/HumanTimeDiffWithDateTip";
import Icon from "components/Icon/Icon";

import ScriptBatchSummaryModal from "pages/DashboardPage/cards/ActivityFeed/components/ScriptBatchSummaryModal";
import { IScriptBatchDetailsForSummary } from "pages/DashboardPage/cards/ActivityFeed/components/ScriptBatchSummaryModal/ScriptBatchSummaryModal";

import { IScriptsCommonProps } from "../../ScriptsNavItems";
import { ScriptsLocation } from "../../Scripts";

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

export type IScriptBatchProgressProps = IScriptsCommonProps & {
  location?: ScriptsLocation;
};

const ScriptBatchProgress = ({
  location,
  router,
  teamId,
}: IScriptBatchProgressProps) => {
  const [
    batchDetailsForSummary,
    setShowBatchDetailsForSummary,
  ] = useState<IScriptBatchDetailsForSummary | null>(null);
  const [batchCount, setBatchCount] = useState<number | null>(null);
  const [updating, setUpdating] = useState(false);

  const statusParam = location?.query.status;

  const selectedStatus = statusParam as ScriptBatchStatus;

  const queryClient = useQueryClient();
  const DEFAULT_PAGE_SIZE = 10;

  const fetchPage = useCallback(
    (pageNumber: number) => {
      setUpdating(true);
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
            })
            .finally(() => {
              setUpdating(false);
            });
        },
        {
          staleTime: 100,
        }
      );
    },
    [queryClient, selectedStatus, teamId]
  );

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

      setBatchCount(null);
      setUpdating(false);
    },
    [location?.search, router]
  );

  const onClickRow = (r: IScriptBatchSummaryV2) => {
    setShowBatchDetailsForSummary({
      batch_execution_id: r.batch_execution_id,
      script_name: r.script_name,
      host_count: r.targeted_host_count,
    });
    // return satisfies caller expectations, not used in this case
    return r;
  };

  const getWhen = (summary: IScriptBatchSummaryV2) => {
    const {
      batch_execution_id: id,
      not_before,
      started_at,
      finished_at,
      canceled,
    } = summary;
    switch (summary.status) {
      case "started":
        if (!started_at || !isDateTimePast(started_at)) {
          console.warn(
            `Batch run with execution id ${id} is marked as 'started' but has no past 'started_at'`
          );
          return null;
        }
        return (
          <>
            Started{" "}
            <HumanTimeDiffWithFleetLaunchCutoff
              timeString={started_at}
              tooltipPosition="right"
            />
          </>
        );
      case "scheduled":
        if (!not_before || isDateTimePast(not_before)) {
          console.warn(
            `Batch run with execution id ${id} is marked as 'scheduled' but has no future scheduled start time`
          );
          return null;
        }
        return (
          <>
            Scheduled to start{" "}
            <HumanTimeDiffWithFleetLaunchCutoff
              timeString={not_before}
              tooltipPosition="right"
            />
          </>
        );
      case "finished":
        if (!finished_at || !isDateTimePast(finished_at)) {
          console.warn(
            `Batch run with execution id ${id} is marked as 'finished' but has no past 'finished_at' data`
          );
          return null;
        }
        return (
          <>
            <Icon
              name={canceled ? "close-filled" : "success"}
              color="ui-fleet-black-50"
              size="small"
            />
            {canceled ? "Canceled" : "Completed"}
            <HumanTimeDiffWithFleetLaunchCutoff
              timeString={finished_at}
              tooltipPosition="right"
            />
          </>
        );
      default:
        return null;
    }
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

  // Fetch the first page of the list when first visiting a tab.
  useEffect(() => {
    if (batchCount === null && !updating) {
      fetchPage(0);
    }
  }, [batchCount, updating, fetchPage]);

  // Reset to first tab if status is invalid.
  useEffect(() => {
    if (!isValidScriptBatchStatus(statusParam)) {
      handleTabChange(0);
    }
  }, [statusParam, handleTabChange]);

  const renderTabContent = (status: ScriptBatchStatus) => {
    // If we're switching to a new tab, show the loading spinner
    // while we get the first page and # of results.
    if (updating && batchCount === null) {
      return (
        <div className={`${baseClass}__loading`}>
          <Spinner />
        </div>
      );
    }

    if (batchCount === 0) {
      return getEmptyState(status);
    }

    return (
      <div className={`${baseClass}__tab-content`}>
        {!updating && batchCount && (
          <div className={`${baseClass}__status-count`}>
            {batchCount} batch script{batchCount > 1 ? "s" : ""}
          </div>
        )}
        <PaginatedList<IScriptBatchSummaryV2>
          count={batchCount || 0}
          fetchPage={fetchPage}
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
      {batchDetailsForSummary && (
        <ScriptBatchSummaryModal
          scriptBatchExecutionDetails={{ ...batchDetailsForSummary }}
          onCancel={() => {
            setShowBatchDetailsForSummary(null);
          }}
          router={router}
        />
      )}
    </>
  );
};

export default ScriptBatchProgress;
