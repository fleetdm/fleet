import { combineReducers } from 'redux';
import invites from './invites/reducer';
import users from './users/reducer';

export default combineReducers({
  invites,
  users,
});
