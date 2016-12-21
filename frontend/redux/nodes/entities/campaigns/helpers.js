export const destroyFunc = (campaign) => {
  return Promise.resolve(campaign);
};

export const updateFunc = (campaign, socketData) => {
  return new Promise((resolve, reject) => {
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
      const newQueryResults = rows.map((row) => {
        return { ...row, hostname: host.hostname };
      });

      return resolve({
        ...campaign,
        hosts: [
          ...hosts,
          host,
        ],
        query_results: [
          ...queryResults,
          ...newQueryResults,
        ],
      });
    }

    return reject();
  });
};

export default { destroyFunc, updateFunc };
