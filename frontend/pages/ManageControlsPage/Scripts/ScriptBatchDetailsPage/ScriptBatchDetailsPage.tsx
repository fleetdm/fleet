import React, {
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useQuery } from "react-query";
import { RouteComponentProps } from "react-router";
import { AxiosError } from "axios";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import { buildQueryStringFromParams } from "utilities/url";

import { NotificationContext } from "context/notification";

import scriptsAPI, {
  IScriptBatchSummaryQueryKey,
  IScriptBatchSummaryV2,
} from "services/entities/scripts";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import {
  isValidScriptBatchHostStatus,
  ScriptBatchHostStatus,
} from "interfaces/script";

import paths from "router/paths";

import ScriptDetailsModal from "pages/hosts/components/ScriptDetailsModal";

import BackLink from "components/BackLink";
import MainContent from "components/MainContent";
import SectionHeader from "components/SectionHeader";
import Spinner from "components/Spinner";
import ActionButtons from "components/buttons/ActionButtons/ActionButtons";
import DataError from "components/DataError";
import TabNav from "components/TabNav";
import TabText from "components/TabText";

import getWhen from "../helpers";
import CancelScriptBatchModal from "../components/CancelScriptBatchModal";

const baseClass = "script-batch-details-page";

export const EMPTY_STATE_DETAILS: Record<ScriptBatchHostStatus, string> = {
  ran: "Hosts with successful script results appear here.",
  errored: "Hosts with error results appear here. ",
  pending: "Compatible hosts that haven't run the script appear here.",
  incompatible:
    "Targeted hosts with incompatible operating systems appear here.",
  canceled: "Hosts where this script run was cancelled appear here.",
};

const getEmptyState = (status: ScriptBatchHostStatus) => {
  return (
    <div className={`${baseClass}__empty`}>
      <b>No hosts with this status</b>
      <p>{EMPTY_STATE_DETAILS[status]}</p>
    </div>
  );
};

const STATUS_BY_INDEX: ScriptBatchHostStatus[] = [
  "ran",
  "errored",
  "pending",
  "incompatible",
  "canceled",
];

interface IScriptBatchDetailsRouteParams {
  batch_execution_id: string;
}

type IScriptBatchDetailsProps = RouteComponentProps<
  undefined,
  IScriptBatchDetailsRouteParams
>;

const ScriptBatchDetailsPage = ({
  router,
  routeParams,
  location,
}: IScriptBatchDetailsProps) => {
  const { batch_execution_id: batchExecutionId } = routeParams;
  const statusParam = location?.query.status;
  const selectedStatus = statusParam as ScriptBatchHostStatus;

  const [showCancelModal, setShowCancelModal] = useState(false);
  const [showScriptDetails, setShowScriptDetails] = useState(false);
  const [isCanceling, setIsCanceling] = useState(false);

  const { renderFlash } = useContext(NotificationContext);

  const { data: batchDetails, isLoading, isError } = useQuery<
    IScriptBatchSummaryV2,
    AxiosError,
    IScriptBatchSummaryV2,
    IScriptBatchSummaryQueryKey[]
  >(
    [{ scope: "script_batch_summary", batch_execution_id: batchExecutionId }],
    ({ queryKey }) => scriptsAPI.getRunScriptBatchSummaryV2(queryKey[0]),
    { ...DEFAULT_USE_QUERY_OPTIONS, enabled: !!batchExecutionId }
  );

  const pathToProgress = useMemo(() => {
    const params = buildQueryStringFromParams({
      status: batchDetails?.status,
      team_id: batchDetails?.team_id,
    });

    return paths.CONTROLS_SCRIPTS_BATCH_PROGRESS + (params ? `?${params}` : "");
  }, [batchDetails?.status, batchDetails?.team_id]);

  const onCancelBatch = useCallback(async () => {
    setIsCanceling(true);

    try {
      await scriptsAPI.cancelScriptBatch(batchExecutionId);
      renderFlash(
        "success",
        <span className={`${baseClass}__success-message`}>
          <span>Successfully canceled script.</span>
        </span>
      );
      setShowCancelModal(false);
      router.push(pathToProgress);
    } catch (error) {
      renderFlash("error", "Could not cancel script. Please try again.");
    } finally {
      setIsCanceling(false);
    }
  }, [batchExecutionId, pathToProgress, renderFlash, router]);

  const handleTabChange = useCallback(
    (index: number) => {
      const newStatus = STATUS_BY_INDEX[index];

      const newParams = new URLSearchParams(location?.search);
      newParams.set("status", newStatus);
      const newQuery = newParams.toString();

      router.push(
        paths
          .CONTROLS_SCRIPTS_BATCH_DETAILS(batchExecutionId)
          .concat(newQuery ? `?${newQuery}` : "")
      );
    },
    [batchExecutionId, location?.search, router]
  );

  // Reset to first tab if status is invalid.
  useEffect(() => {
    if (!isValidScriptBatchHostStatus(selectedStatus)) {
      handleTabChange(0);
    }
  }, [handleTabChange, selectedStatus]);

  // const renderTabContent = (status: ScriptBatchHostStatus, statusCount: number) => {
  const renderTabContent = ([status, statusCount]: [
    ScriptBatchHostStatus,
    number
  ]) => {
    if (statusCount === 0) {
      return getEmptyState(status);
    }
    return (
      <ScriptBatchHostsTable
        batchExecutionId={batchExecutionId}
        status={status}
      />
    );
  };

  const renderContent = () => {
    if (isLoading || !batchDetails) {
      return <Spinner />;
    }
    if (isError) {
      return <DataError description="Could not load script batch details." />;
    }
    const {
      script_name,
      status,

      targeted_host_count: targeted,
      ran_host_count: ran,
      errored_host_count: errored,
      pending_host_count: pending,
      incompatible_host_count: incompatible,
      canceled_host_count: canceled,
    } = batchDetails || {};

    const getStatusAndCountByIndex = (i: number) =>
      ([
        ["ran", ran],
        ["errored", errored],
        ["pending", pending],
        ["incompatible", incompatible],
        ["canceled", canceled],
      ] as [ScriptBatchHostStatus, number][])[i];

    const subTitle = (
      <>
        <span>
          <b>{targeted}</b> hosts targeted (
          {Math.ceil((ran + errored) / targeted)}% responded)
        </span>
        <span className="when">{getWhen(batchDetails)}</span>
      </>
    );

    return (
      <div className={`${baseClass}`}>
        <BackLink text="Back to script activity" path={pathToProgress} />

        <SectionHeader
          wrapperCustomClass={`${baseClass}__header`}
          title={script_name}
          subTitle={subTitle}
          details={
            <ActionButtons
              baseClass={baseClass}
              actions={[
                {
                  type: "secondary",
                  label: "Show script",
                  buttonVariant: "text-icon",
                  iconName: "eye",
                  onClick: () => {
                    setShowScriptDetails(true);
                  },
                },
                {
                  type: "secondary",
                  label: "Cancel",
                  onClick: () => {
                    setShowCancelModal(true);
                  },
                  hideAction: status === "finished",
                  buttonVariant: "alert",
                },
              ]}
            />
          }
          alignLeftHeaderVertically
          greySubtitle
        />
        <TabNav>
          <Tabs
            selectedIndex={STATUS_BY_INDEX.indexOf(selectedStatus)}
            onSelect={handleTabChange}
          >
            <TabList>
              <Tab>
                <TabText count={ran}>Ran</TabText>
              </Tab>
              <Tab>
                <TabText count={errored} redCount>
                  Errored
                </TabText>
              </Tab>
              <Tab>
                <TabText count={pending} greyCount>
                  Pending
                </TabText>
              </Tab>
              <Tab>
                <TabText count={incompatible} greyCount>
                  Incompatible
                </TabText>
              </Tab>
              <Tab>
                <TabText count={canceled} greyCount>
                  Canceled
                </TabText>
              </Tab>
            </TabList>
            <TabPanel>{renderTabContent(getStatusAndCountByIndex(0))}</TabPanel>
            <TabPanel>{renderTabContent(getStatusAndCountByIndex(1))}</TabPanel>
            <TabPanel>{renderTabContent(getStatusAndCountByIndex(2))}</TabPanel>
            <TabPanel>{renderTabContent(getStatusAndCountByIndex(3))}</TabPanel>
            <TabPanel>{renderTabContent(getStatusAndCountByIndex(4))}</TabPanel>
          </Tabs>
        </TabNav>
      </div>
    );
  };

  return (
    <>
      <MainContent className={baseClass}>{renderContent()}</MainContent>
      {showCancelModal && (
        <CancelScriptBatchModal
          onSubmit={onCancelBatch}
          onExit={() => {
            setShowCancelModal(false);
          }}
          scriptName={batchDetails?.script_name}
          isCanceling={isCanceling}
        />
      )}
      {showScriptDetails && (
        <ScriptDetailsModal
          selectedScriptId={batchDetails?.script_id}
          onCancel={() => {
            setShowScriptDetails(false);
          }}
          suppressSecondaryActions
        />
      )}
    </>
  );
};

export default ScriptBatchDetailsPage;
