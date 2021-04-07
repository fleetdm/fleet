import endpoints from 'kolide/endpoints';

// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Base from 'kolide/base';

export default (client: any) => {
  return {
    create: () => {
      return {};
    },

    loadAll: () => {
      return [];
    },

    update: () => {
      return {};
    },
  };
};
