import { ICampaign, ICampaignState } from "interfaces/campaign";
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
    errors = [],
    hosts = [],
    hosts_count: hostsCount = { total: 0, failed: 0, successful: 0 },
    query_results: queryResults = [],
  } = campaign;
  const { error, host, rows = [] } = data;

  let newErrors;
  let newHosts;
  let newHostsCount;

  if (error || error === "") {
    const newFailed = hostsCount.failed + 1;
    const newTotal = hostsCount.successful + newFailed;

    newErrors = errors.concat([
      {
        host_display_name: host?.display_name,
        osquery_version: host?.osquery_version,
        error:
          error ||
          // Hosts with osquery version below 4.4.0 receive an empty error message
          // when the live query fails so we create our own message.
          "Error details require osquery 4.4.0+ (Launcher does not provide error details)",
      },
    ]);
    newHostsCount = {
      successful: hostsCount.successful,
      failed: newFailed,
      total: newTotal,
    };
    newHosts = hosts;
  } else {
    const newSuccessful = hostsCount.successful + 1;
    const newTotal = hostsCount.failed + newSuccessful;

    newErrors = [...errors];
    newHostsCount = {
      successful: newSuccessful,
      failed: hostsCount.failed,
      total: newTotal,
    };
    const newHost = { ...host, query_results: rows };
    newHosts = hosts.concat(newHost);
  }

  return {
    campaign: {
      ...campaign,
      errors: newErrors,
      hosts: newHosts,
      hosts_count: newHostsCount,
      query_results: [...queryResults, ...rows],
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
