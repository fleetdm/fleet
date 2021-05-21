// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Kolide from "kolide";
// @ts-ignore
import Config from "redux/nodes/entities/base/config";
// @ts-ignore
import schemas from "redux/nodes/entities/base/schemas";
// @ts-ignore
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
