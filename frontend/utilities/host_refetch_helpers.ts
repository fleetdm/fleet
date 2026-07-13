import { IHost } from "interfaces/host";

type IRefetchHost = Pick<
  IHost,
  "seen_time" | "distributed_interval" | "config_tls_refresh"
>;

// Matches the buffer the backend adds to a host's check-in interval before
// considering it offline (see OnlineIntervalBuffer in server/fleet/hosts.go),
// so our "give up" window lines up with the same online/offline threshold.
const ONLINE_INTERVAL_BUFFER_SECONDS = 60;

// How many of the host's own check-in cycles to wait before giving up on a
// refetch, so a host with a long distributed_interval isn't cut off before
// it's even had a chance to check in once.
const CHECKIN_CYCLES_BEFORE_GIVING_UP = 2;

/** Mirrors the server's onlineInterval calculation: the shorter of the
 * host's two check-in intervals, plus a flapping buffer, in ms. */
export const getExpectedCheckInIntervalMs = (host: IRefetchHost): number => {
  const intervalSeconds = Math.min(
    host.distributed_interval,
    host.config_tls_refresh
  );
  return (intervalSeconds + ONLINE_INTERVAL_BUFFER_SECONDS) * 1000;
};

/** How long to poll before giving up on a refetch, adapted to the host's own
 * check-in cadence but never shorter than `fallbackMs`. */
export const getRefetchGiveUpDelayMs = (
  host: IRefetchHost,
  fallbackMs: number
): number =>
  Math.max(
    fallbackMs,
    getExpectedCheckInIntervalMs(host) * CHECKIN_CYCLES_BEFORE_GIVING_UP
  );

export type RefetchGiveUpReason = "checkin_stalled" | "refetch_stalled";

/**
 * Once we've given up polling, decide whether the host simply hasn't
 * checked in during the wait (no evidence anything is wrong with refetch
 * itself) or whether it checked in but `refetch_requested` never cleared
 * (a real anomaly worth surfacing as an error).
 */
export const getRefetchGiveUpReason = (
  host: Pick<IHost, "seen_time">,
  refetchStartTime: number
): RefetchGiveUpReason =>
  new Date(host.seen_time).getTime() > refetchStartTime
    ? "refetch_stalled"
    : "checkin_stalled";
