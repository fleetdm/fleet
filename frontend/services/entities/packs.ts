import sendRequest from "services";
import { omit } from "lodash";

import endpoints from "fleet/endpoints";
import { IPack } from "interfaces/pack";
import { ITargets } from "interfaces/target";
import helpers from "fleet/helpers";

interface ICreateProps {
  name: string;
  description: string;
  targets: ITargets;
}

export default {
  addLabel: (packID: number, labelID: number) => {
    const { PACKS } = endpoints;
    const path = `${PACKS}/${packID}/labels/${labelID}`;

    return sendRequest("POST", path); // type of request, path, info (optional)
  },
  addQuery: (packID: number, queryID: number) => {
    const { PACKS } = endpoints;
    const path = `${PACKS}/${packID}/queries/${queryID}`;

    return sendRequest("POST", path);
  },
  create: ({ name, description, targets }: ICreateProps) => {
    const { PACKS } = endpoints;
    const packTargets = helpers.formatSelectedTargetsForApi(targets, true);

    return sendRequest("POST", PACKS, { name, description, ...packTargets }); // sendRequest /axios stringifies for you
  }, // do not need .then((response)), can pull response.pack out on page
  destroy: (id: number) => {
    const { PACKS } = endpoints;
    const path = `${PACKS}/id/${id}`;

    return sendRequest("DELETE", path);
  },
  load: (packID: number) => {
    const { PACKS } = endpoints;
    const path = `${PACKS}/${packID}`; // this had client.baseURL on it... already built into send request in services/index

    return sendRequest("GET", path);
  },
  loadAll: () => {
    const { PACKS } = endpoints;

    return sendRequest("GET", PACKS);
  },
  update: (pack: IPack, updatedPack: any) => {
    // TODO: fix any later
    const { PACKS } = endpoints;
    const { targets } = updatedPack;
    const path = `${PACKS}/${pack.id}`;

    let packTargets = null;
    if (targets) {
      packTargets = helpers.formatSelectedTargetsForApi(targets, true);
    }

    const packWithoutTargets = omit(updatedPack, "targets");
    const packParams = { ...packWithoutTargets, ...packTargets };

    return sendRequest("PATCH", path, packParams);
  },
};
