import configAPI from "services/entities/config";
import { IConfig } from "interfaces/config";

// cached_mysql in the backend caches AppConfig for 1s per process. After a
// state-changing call (Android MDM turn on/off) returns, other replicas (and
// occasionally the same replica) may briefly serve a stale config. Poll until
// the expected state is visible, then fall through after a bounded number of
// attempts so the UI never hangs.
const refetchConfigUntil = async (
  predicate: (config: IConfig) => boolean,
  attempts = 4,
  delayMs = 300
): Promise<IConfig> => {
  const config = await configAPI.loadAll();
  if (predicate(config) || attempts <= 1) {
    return config;
  }
  await new Promise((resolve) => {
    setTimeout(resolve, delayMs);
  });
  return refetchConfigUntil(predicate, attempts - 1, delayMs);
};

export default refetchConfigUntil;
