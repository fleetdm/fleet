import Kolide from "kolide";
import Config from "redux/nodes/entities/base/config";
import schemas from "redux/nodes/entities/base/schemas";
import { parseEntityFunc } from "redux/nodes/entities/hosts/helpers";

const { HOSTS: schema } = schemas;

export default new Config({
  destroyFunc: Kolide.hosts.destroy,
  entityName: "hosts",
  loadAllFunc: Kolide.hosts.loadAll,
  loadFunc: Kolide.hosts.load,
  parseEntityFunc,
  schema,
});
