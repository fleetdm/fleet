export const destroyFunc = (campaign) => {
  return Promise.resolve(campaign);
};

export const update = (campaign, socketData) => {
  return new Promise((resolve) => {
    const { type, data } = socketData;

    if (type === 'totals') {
      return resolve({
        ...campaign,
        totals: data,
      });
    }

    if (type === 'result') {
      const queryResults = campaign.query_results || [];
      const hosts = campaign.hosts || [];
      const { host, rows } = data;
      const { hosts_count: hostsCount } = campaign;
      let newHostsCount;

      if (data.error) {
        const newFailed = hostsCount.failed + 1;
        const newTotal = hostsCount.successful + newFailed;

        newHostsCount = {
          successful: hostsCount.successful,
          failed: newFailed,
          total: newTotal,
        };
      } else {
        const newSuccessful = hostsCount.successful + 1;
        const newTotal = hostsCount.failed + newSuccessful;

        newHostsCount = {
          successful: newSuccessful,
          failed: hostsCount.failed,
          total: newTotal,
        };
      }

      return resolve({
        ...campaign,
        hosts: [
          ...hosts,
          host,
        ],
        query_results: [
          ...queryResults,
          ...rows,
        ],
        hosts_count: newHostsCount,
      });
    }

    if (type === 'status') {
      const { status } = data;

      return resolve({ ...campaign, status });
    }

    return resolve(campaign);
  });
};

export default { destroyFunc, update };
