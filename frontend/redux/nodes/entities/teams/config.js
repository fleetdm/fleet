import Kolide from 'kolide';
import Config from 'redux/nodes/entities/base/config';
import schemas from 'redux/nodes/entities/base/schemas';

const { TEAMS } = schemas;

export default new Config({
  createFunc: Kolide.teams.create,
  entityName: 'teams',
  loadAllFunc: Kolide.teams.loadAll,
  schema: TEAMS,
  updateFunc: Kolide.teams.update,
});
