import { setupServer } from "msw/node";

import handlers from "./default-handlers";

const mockServer = setupServer(...handlers);

export default mockServer;
