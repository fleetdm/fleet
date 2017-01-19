import { filter, includes } from 'lodash';
import moment from 'moment';

const filterHosts = (hosts, label) => {
  if (!label) {
    return hosts;
  }

  if (label.id === 'new') {
    return filter(hosts, h => moment().diff(h.created_at, 'hours') <= 24);
  }

  const { host_ids: hostIDs, platform, slug, type } = label;

  switch (type) {
    case 'all':
      return hosts;
    case 'status':
      return filter(hosts, { status: slug });
    case 'platform':
      return filter(hosts, { platform });
    case 'custom':
      return filter(hosts, h => includes(hostIDs, h.id));
    default:
      return hosts;
  }
};

export default { filterHosts };
