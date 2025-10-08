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
import RunScriptDetailsModal from "pages/DashboardPage/cards/ActivityFeed/components/RunScriptDetailsModal";

import BackButton from "components/BackButton";
import MainContent from "components/MainContent";
import SectionHeader from "components/SectionHeader";
import Spinner from "components/Spinner";
import ActionButtons from "components/buttons/ActionButtons/ActionButtons";
import DataError from "components/DataError";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import ViewAllHostsLink from "components/ViewAllHostsLink";

import getWhen from "../helpers";
import CancelScriptBatchModal from "../components/CancelScriptBatchModal";
import ScriptBatchHostsTable from "./components/ScriptBatchHostsTable";

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

const HOSTS_STATUS_BY_INDEX: ScriptBatchHostStatus[] = [
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

  const hostStatusParam = location.query.status;
  const pageParam = parseInt(location.query.page ?? "0", 10);
  const orderKeyParam = location.query.order_key ?? "display_name";
  const orderDirectionParam = location.query.order_direction ?? "asc";

  const selectedHostStatus = hostStatusParam as ScriptBatchHostStatus;

  const [showCancelModal, setShowCancelModal] = useState(false);
  const [showBatchScriptDetails, setShowBatchScriptDetails] = useState(false);
  const [
    hostScriptExecutionIdForModal,
    setHostScriptExecutionIdForModal,
  ] = useState<string | null>(null);
  const [isCanceling, setIsCanceling] = useState(false);

  const { renderFlash } = useContext(NotificationContext);

  const {
    data: batchDetails,
    isLoading,
    isError,
    refetch: refetchBatchDetails,
  } = useQuery<
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
      renderFlash("success", "Successfully canceled script.");
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
      const newHostsStatus = HOSTS_STATUS_BY_INDEX[index];

      const newParams = new URLSearchParams(location?.search);
      newParams.set("status", newHostsStatus);
      newParams.set("page", "0");
      const newQuery = newParams.toString();

      router.push(
        paths
          .CONTROLS_SCRIPTS_BATCH_DETAILS(batchExecutionId)
          .concat(newQuery ? `?${newQuery}` : "")
      );
      // update page's summary data (e.g. pct hosts responded) whenever changing tabs
      refetchBatchDetails();
    },
    [batchExecutionId, location?.search, refetchBatchDetails, router]
  );

  useEffect(() => {
    if (!isValidScriptBatchHostStatus(selectedHostStatus)) {
      handleTabChange(0);
    }
  }, [handleTabChange, selectedHostStatus]);

  const renderTabContent = ([hostStatus, hostStatusCount]: [
    ScriptBatchHostStatus,
    number
  ]) => {
    if (hostStatusCount === 0) {
      return getEmptyState(hostStatus);
    }
    return (
      <div className={`${baseClass}__tab-content`}>
        <span className={`${baseClass}__tab-content__header`}>
          <b>
            {hostStatusCount} host{hostStatusCount > 1 && "s"}
          </b>
          <ViewAllHostsLink
            queryParams={{
              script_batch_execution_status: selectedHostStatus, // refers to script batch host status, may update pending conv w Rachael
              script_batch_execution_id: batchExecutionId,
              team_id: batchDetails?.team_id,
            }}
          />
        </span>
        <ScriptBatchHostsTable
          batchExecutionId={batchExecutionId}
          selectedHostStatus={hostStatus}
          page={pageParam}
          orderDirection={orderDirectionParam}
          orderKey={orderKeyParam}
          setHostScriptExecutionIdForModal={setHostScriptExecutionIdForModal}
          router={router}
        />
      </div>
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

    const getHostStatusAndCountByIndex = (i: number) =>
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
          {Math.ceil(100 * ((ran + errored) / targeted))}% responded)
        </span>
        <span className="when">{getWhen(batchDetails)}</span>
      </>
    );

    return (
      <>
        <div className={`${baseClass}__header-links`}>
          <BackButton text="Back to script activity" path={pathToProgress} />
        </div>
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
                  buttonVariant: "inverse",
                  iconName: "eye",
                  onClick: () => {
                    setShowBatchScriptDetails(true);
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
            selectedIndex={HOSTS_STATUS_BY_INDEX.indexOf(selectedHostStatus)}
            onSelect={handleTabChange}
          >
            <TabList>
              <Tab>
                <TabText>Ran</TabText>
              </Tab>
              <Tab>
                <TabText>Errored</TabText>
              </Tab>
              <Tab>
                <TabText>Pending</TabText>
              </Tab>
              <Tab>
                <TabText>Incompatible</TabText>
              </Tab>
              <Tab>
                <TabText>Canceled</TabText>
              </Tab>
            </TabList>
            <TabPanel>
              {renderTabContent(getHostStatusAndCountByIndex(0))}
            </TabPanel>
            <TabPanel>
              {renderTabContent(getHostStatusAndCountByIndex(1))}
            </TabPanel>
            <TabPanel>
              {renderTabContent(getHostStatusAndCountByIndex(2))}
            </TabPanel>
            <TabPanel>
              {renderTabContent(getHostStatusAndCountByIndex(3))}
            </TabPanel>
            <TabPanel>
              {renderTabContent(getHostStatusAndCountByIndex(4))}
            </TabPanel>
          </Tabs>
        </TabNav>
      </>
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
      {showBatchScriptDetails && (
        <ScriptDetailsModal
          selectedScriptId={batchDetails?.script_id}
          onCancel={() => {
            setShowBatchScriptDetails(false);
          }}
          suppressSecondaryActions
        />
      )}
      {hostScriptExecutionIdForModal && (
        <RunScriptDetailsModal
          scriptExecutionId={hostScriptExecutionIdForModal}
          onCancel={() => setHostScriptExecutionIdForModal(null)}
        />
      )}
    </>
  );
};

export default ScriptBatchDetailsPage;
