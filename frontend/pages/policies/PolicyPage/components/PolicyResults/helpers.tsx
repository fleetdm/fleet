import { IPolicyHostResponse } from "interfaces/host";

export const getYesNoCounts = (hostResponses: IPolicyHostResponse[]) => {
  const yesNoCounts = hostResponses.reduce(
    (acc, hostResponse) => {
      if (hostResponse.query_results?.length) {
        acc.yes += 1;
      } else {
        acc.no += 1;
      }
      return acc;
    },
    { yes: 0, no: 0 }
  );

  return yesNoCounts;
};

export default { getYesNoCounts };
