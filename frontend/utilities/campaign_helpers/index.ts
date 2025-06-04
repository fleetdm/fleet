import {
  ICampaign,
  ICampaignState,
  IUIHostCounts,
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

interface IIncomingCampaignStatus {
  type: "status";
  data: {
    // acutal_results == count_of_hosts_with_results + count_of_hosts_with_no_results
    actual_results: number;
    count_of_hosts_with_results: number;
    count_of_hosts_with_no_results: number;
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

type ISocketData = IResult | IIncomingCampaignStatus | ITotals | IError;

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
    uiHostCounts: prevUIHostCounts = { total: 0, failed: 0, successful: 0 },
    queryResults: prevQueryResults = [],
  } = campaign;
  const { error: curError, host: curHost, rows: curQueryResults = [] } = data;

  let newErrors;
  let newHostsWithResults: IHostWithQueryResults[];
  // uiHostCounts.total is incremented by 1 in both cases, error and results
  let newUIHostCounts: IUIHostCounts;

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
    newUIHostCounts = {
      total: prevUIHostCounts.total + 1,
      successful: prevUIHostCounts.successful,
      failed: prevUIHostCounts.failed + 1,
    };
    newHostsWithResults = prevHostsWithResults;
  } else {
    // received non-error response
    newErrors = [...prevErrors];
    newUIHostCounts = {
      total: prevUIHostCounts.total + 1,
      successful: prevUIHostCounts.successful + 1,
      failed: prevUIHostCounts.failed,
    };
    const curHostWithResults = { ...curHost, query_results: curQueryResults };
    newHostsWithResults = prevHostsWithResults.concat(curHostWithResults);
  }

  return {
    campaign: {
      ...campaign,
      errors: newErrors,
      hosts: newHostsWithResults,
      uiHostCounts: newUIHostCounts,
      queryResults: [...prevQueryResults, ...curQueryResults],
    },
  };
};

const updateCampaignStateFromStatus = (
  prevCampaign: ICampaign,
  {
    data: {
      status,
      count_of_hosts_with_results: newCountOfHostsWithResults,
      count_of_hosts_with_no_results: newCountOfHostsWithNoResults,
    },
  }: IIncomingCampaignStatus
) => {
  return {
    campaign: {
      ...prevCampaign,
      status,
      serverHostCounts: {
        countOfHostsWithResults: newCountOfHostsWithResults,
        countOfHostsWithNoResults: newCountOfHostsWithNoResults,
      },
    },
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
