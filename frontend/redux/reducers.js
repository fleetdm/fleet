import { combineReducers } from 'redux';
import { routerReducer } from 'react-router-redux';

import app from './nodes/app/reducer';
import auth from './nodes/auth/reducer';
import components from './nodes/components/reducer';
import entities from './nodes/entities/reducer';
import notifications from './nodes/notifications/reducer';
import redirectLocation from './nodes/redirectLocation/reducer';

export default combineReducers({
  app,
  auth,
  components,
  entities,
  notifications,
  redirectLocation,
  routing: routerReducer,
});
