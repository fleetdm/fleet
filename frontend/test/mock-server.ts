import { setupServer } from "msw/node";

import handlers from "./server-handlers";

const mockServer = setupServer(...handlers);

export default mockServer;
