/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
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

    return sendRequest("POST", path);
  },
  addQuery: (packID: number, queryID: number) => {
    const { PACKS } = endpoints;
    const path = `${PACKS}/${packID}/queries/${queryID}`;

    return sendRequest("POST", path);
  },
  create: ({ name, description, targets }: ICreateProps) => {
    const { PACKS } = endpoints;
    const packTargets = helpers.formatSelectedTargetsForApi(targets, true);

    return sendRequest("POST", PACKS, { name, description, ...packTargets });
  },
  destroy: (packID: number) => {
    const { PACKS } = endpoints;
    const path = `${PACKS}/id/${packID}`;

    return sendRequest("DELETE", path);
  },
  load: (packID: number) => {
    const { PACKS } = endpoints;
    const path = `${PACKS}/${packID}`;

    return sendRequest("GET", path);
  },
  loadAll: () => {
    const { PACKS } = endpoints;

    return sendRequest("GET", PACKS);
  },
  update: (packId: number, updatedPack: any) => {
    const { PACKS } = endpoints;
    const { targets } = updatedPack;
    const path = `${PACKS}/${packId}`;

    let packTargets = null;
    if (targets) {
      packTargets = helpers.formatSelectedTargetsForApi(targets, true);
    }

    const packWithoutTargets = omit(updatedPack, "targets");
    const packParams = { ...packWithoutTargets, ...packTargets };

    return sendRequest("PATCH", path, packParams);
  },
};
