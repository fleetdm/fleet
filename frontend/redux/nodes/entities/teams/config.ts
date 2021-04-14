// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Kolide from 'kolide';
// @ts-ignore
import Config from 'redux/nodes/entities/base/config';
// @ts-ignore
import schemas from 'redux/nodes/entities/base/schemas';

const { TEAMS } = schemas;

export default new Config({
  createFunc: Kolide.teams.create,
  destroyFunc: Kolide.teams.destroy,
  entityName: 'teams',
  loadAllFunc: Kolide.teams.loadAll,
  schema: TEAMS,
  updateFunc: Kolide.teams.update,
});
