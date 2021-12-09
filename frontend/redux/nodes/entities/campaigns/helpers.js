export const destroyFunc = (campaign) => {
  return Promise.resolve(campaign);
};

const updateCampaignStateFromTotals = (campaign, { data }) => {
  return {
    campaign: { ...campaign, totals: data },
  };
};

const updateCampaignStateFromResults = (campaign, { data }) => {
  const queryResults = campaign.query_results || [];
  const errors = campaign.errors || [];
  const hosts = campaign.hosts || [];
  const { host, rows, error } = data;
  host.query_results = rows;
  const { hosts_count: hostsCount } = campaign;
  const newHosts = [...hosts, host];
  const newQueryResults = [...queryResults, ...rows];
  let newHostsCount;
  let newErrors;
  // Host's with osquery version above 4.4.0 receive an error message
  // when the live query fails.
  if (error) {
    const newFailed = hostsCount.failed + 1;
    const newTotal = hostsCount.successful + newFailed;

    newHostsCount = {
      successful: hostsCount.successful,
      failed: newFailed,
      total: newTotal,
    };

    newErrors = [
      ...errors,
      {
        host_hostname: host.hostname,
        osquery_version: host.osquery_version,
        error,
      },
    ];
    // Host's with osquery version below 4.4.0 receive an empty error message
    // when the live query fails so we create our own message.
  } else if (error === "") {
    const newFailed = hostsCount.failed + 1;
    const newTotal = hostsCount.successful + newFailed;

    newHostsCount = {
      successful: hostsCount.successful,
      failed: newFailed,
      total: newTotal,
    };
    newErrors = [
      ...errors,
      {
        host_hostname: host.hostname,
        osquery_version: host.osquery_version,
        error:
          "Error details require osquery 4.4.0+ (Launcher does not provide error details)",
      },
    ];
  } else {
    const newSuccessful = hostsCount.successful + 1;
    const newTotal = hostsCount.failed + newSuccessful;

    newHostsCount = {
      successful: newSuccessful,
      failed: hostsCount.failed,
      total: newTotal,
    };
  }

  return {
    campaign: {
      ...campaign,
      hosts: newHosts,
      query_results: newQueryResults,
      hosts_count: newHostsCount,
      errors: newErrors,
    },
  };
};

const updateCampaignStateFromStatus = (campaign, { data }) => {
  const { status } = data;
  const updatedCampaign = { ...campaign, status };

  return {
    campaign: updatedCampaign,
    queryIsRunning: data !== "finished",
  };
};

export const updateCampaignState = (socketData) => {
  return (prevState) => {
    const { campaign } = prevState;

    switch (socketData.type) {
      case "totals":
        return updateCampaignStateFromTotals(campaign, socketData);
      case "result":
        return updateCampaignStateFromResults(campaign, socketData);
      case "status":
        return updateCampaignStateFromStatus(campaign, socketData);
      default:
        return { campaign };
    }
  };
};

export default { destroyFunc, updateCampaignState };
