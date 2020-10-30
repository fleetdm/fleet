import { combineReducers } from 'redux';
import { loadingBarReducer } from 'react-redux-loading-bar';
import { routerReducer } from 'react-router-redux';

import app from './nodes/app/reducer';
import auth from './nodes/auth/reducer';
import components from './nodes/components/reducer';
import entities from './nodes/entities/reducer';
import notifications from './nodes/notifications/reducer';
import osquery from './nodes/osquery/reducer';
import persistentFlash from './nodes/persistent_flash/reducer';
import redirectLocation from './nodes/redirectLocation/reducer';

export default combineReducers({
  app,
  auth,
  components,
  entities,
  loadingBar: loadingBarReducer,
  notifications,
  osquery,
  persistentFlash,
  redirectLocation,
  routing: routerReducer,
});
