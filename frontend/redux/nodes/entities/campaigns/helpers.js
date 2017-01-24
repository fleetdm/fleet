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
