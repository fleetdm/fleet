/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import { omit } from "lodash";

import endpoints from "utilities/endpoints";
import { formatPackTargetsForApi } from "utilities/helpers";
import { ISelectTargetsEntity } from "interfaces/target";
import { IUpdatePack } from "interfaces/pack";

interface ICreateProps {
  name: string;
  description: string;
  targets: ISelectTargetsEntity[];
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
    const packTargets = formatPackTargetsForApi(targets);

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
  update: (packId: number, updatedPack: IUpdatePack) => {
    const { PACKS } = endpoints;
    const { targets } = updatedPack;
    const path = `${PACKS}/${packId}`;

    let packTargets = null;
    if (targets) {
      packTargets = formatPackTargetsForApi(targets);
    }

    const packWithoutTargets = omit(updatedPack, "targets");
    const packParams = { ...packWithoutTargets, ...packTargets };

    return sendRequest("PATCH", path, packParams);
  },
};
