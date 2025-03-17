import {
  ICampaign,
  ICampaignState,
  IHostCounts,
  IHostWithQueryResults,
} from "interfaces/campaign";
import { IHost } from "interfaces/host";
import { useContext } from "react";
import { NotificationContext } from "context/notification";

interface IResult {
  type: "result";
  data: {
    distributed_query_execution_id: number;
    error: string | null;
    host: IHost;
    rows: Record<string, unknown>[];
  };
}

interface IStatus {
  type: "status";
  data: {
    actual_results: number;
    expected_result: number;
    status: string;
  };
}

interface ITotals {
  type: "totals";
  data: {
    count: number;
    missing_in_action: number;
    offline: number;
    online: number;
  };
}

interface IError {
  type: "error";
  data: string;
}

type ISocketData = IResult | IStatus | ITotals | IError;

const updateCampaignStateFromTotals = (
  campaign: ICampaign,
  { data: totals }: ITotals
) => {
  return {
    campaign: { ...campaign, totals },
  };
};

const updateCampaignStateFromResults = (
  campaign: ICampaign,
  { data }: IResult
) => {
  const {
    errors: prevErrors = [],
    hosts: prevHostsWithResults = [],
    hosts_count: prevHostCounts = { total: 0, failed: 0, successful: 0 },
    query_results: prevQueryResults = [],
  } = campaign;
  const { error: curError, host: curHost, rows: curQueryResults = [] } = data;

  let newErrors;
  let newHostsWithResults: IHostWithQueryResults[];
  // hosts_count.total is incremented by 1 in both cases, error and results
  let newHostCounts: IHostCounts;

  if (curError || curError === "") {
    // both  `campaign.errors` and `campaign.hosts_count.failed` updated by this same condition,
    // therefore `campaign.errors.length` === `campaign.hosts_count.failed`
    newErrors = prevErrors.concat([
      {
        host_display_name: curHost?.display_name,
        osquery_version: curHost?.osquery_version,
        error:
          curError ||
          // Hosts with osquery version below 4.4.0 receive an empty error message
          // when the live query fails so we create our own message.
          "Error details require osquery 4.4.0+ (Launcher does not provide error details)",
      },
    ]);
    newHostCounts = {
      total: prevHostCounts.total + 1,
      successful: prevHostCounts.successful,
      failed: prevHostCounts.failed + 1,
    };
    newHostsWithResults = prevHostsWithResults;
  } else {
    // received non-error response
    newErrors = [...prevErrors];
    newHostCounts = {
      total: prevHostCounts.total + 1,
      successful: prevHostCounts.successful + 1,
      failed: prevHostCounts.failed,
    };
    const curHostWithResults = { ...curHost, query_results: curQueryResults };
    newHostsWithResults = prevHostsWithResults.concat(curHostWithResults);
  }

  return {
    campaign: {
      ...campaign,
      errors: newErrors,
      hosts: newHostsWithResults,
      hosts_count: newHostCounts,
      query_results: [...prevQueryResults, ...curQueryResults],
    },
  };
};

const updateCampaignStateFromStatus = (
  campaign: ICampaign,
  { data: { status } }: IStatus
) => {
  return {
    campaign: { ...campaign, status },
    queryIsRunning: status !== "finished",
  };
};

export const updateCampaignState = (socketData: ISocketData) => {
  return ({ campaign }: ICampaignState) => {
    const { renderFlash } = useContext(NotificationContext);
    switch (socketData.type) {
      case "totals":
        return updateCampaignStateFromTotals(campaign, socketData);
      case "result":
        return updateCampaignStateFromResults(campaign, socketData);
      case "status":
        return updateCampaignStateFromStatus(campaign, socketData);
      case "error":
        if (socketData.data.includes("unexpected exit in receiveMessages")) {
          const campaignID = socketData.data.substring(
            socketData.data.indexOf("=") + 1
          );
          renderFlash(
            "error",
            `Fleet's connection to Redis failed (campaign ID ${campaignID}). If this issue persists, please contact your administrator.`
          );
        }
        return { campaign };
      default:
        return { campaign };
    }
  };
};

export default { updateCampaignState };
