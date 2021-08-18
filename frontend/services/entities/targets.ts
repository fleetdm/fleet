import sendRequest from "services";
import endpoints from "fleet/endpoints";
import { IHost } from "interfaces/host";
import { ILabel } from "interfaces/label";

interface ITargetsProps {
  query?: string;
  queryId?: string | null;
  selected?: {
    hosts: IHost[];
    labels: ILabel[];
  }
}

const defaultSelected = {
  hosts: [],
  labels: [],
};

export default {
  loadAll: ({
    query = "", 
    queryId = null, 
    selected = defaultSelected
  }: ITargetsProps) => {
    const { TARGETS } = endpoints;
  
    return sendRequest("POST", TARGETS, {
      query,
      queryId,
      selected,
    });
  },
}