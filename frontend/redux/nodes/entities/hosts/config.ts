// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Fleet from "fleet";
// @ts-ignore
import Config from "redux/nodes/entities/base/config";
// @ts-ignore
import schemas from "redux/nodes/entities/base/schemas";
// @ts-ignore
import { parseEntityFunc } from "redux/nodes/entities/hosts/helpers";

const { HOSTS: schema } = schemas;

export default new Config({
  destroyFunc: Fleet.hosts.destroy,
  entityName: "hosts",
  loadAllFunc: Fleet.hosts.loadAll,
  loadFunc: Fleet.hosts.load,
  parseEntityFunc,
  schema,
});
