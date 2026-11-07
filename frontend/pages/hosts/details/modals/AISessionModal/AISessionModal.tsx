import { format } from "date-fns";
import React, { useState } from "react";
import { useQuery } from "react-query";

import DataError from "components/DataError";
import EmptyState from "components/EmptyState";
import Icon from "components/Icon";
import InfoBanner from "components/InfoBanner";
import Modal from "components/Modal";
import Pagination from "components/Pagination";
import Spinner from "components/Spinner";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import { IHostAISession } from "interfaces/ai_sessions";
import {
  AI_SESSION_BACKFILL_MESSAGE,
  getAIAgentDisplay,
  getAISessionDuration,
  getAISessionUserAccount,
} from "pages/hosts/details/cards/AISessions/helpers";
import hostAPI, {
  IGetHostAISessionEventsApiParams,
  IGetHostAISessionEventsResponse,
} from "services/entities/hosts";
import {
  DEFAULT_EMPTY_CELL_VALUE,
  DEFAULT_USE_QUERY_OPTIONS,
} from "utilities/constants";
import { syntaxHighlight } from "utilities/helpers";
import stringUtils from "utilities/strings/stringUtils";

const baseClass = "ai-session-modal";

const EVENTS_PAGE_SIZE = 15;

const formatTime = (time: string | null, pattern: string) =>
  time ? format(new Date(time), pattern) : DEFAULT_EMPTY_CELL_VALUE;

interface ISummaryRow {
  id: string;
  value: React.ReactNode;
}

interface IAISessionModalProps {
  hostId: number;
  session: IHostAISession;
  onExit: () => void;
}

const AISessionModal = ({ hostId, session, onExit }: IAISessionModalProps) => {
  const [page, setPage] = useState(0);
  const [expandedEventId, setExpandedEventId] = useState<number | null>(null);

  const { data, isLoading, isPreviousData, isError } = useQuery<
    IGetHostAISessionEventsResponse,
    Error,
    IGetHostAISessionEventsResponse,
    Array<
      IGetHostAISessionEventsApiParams & { scope: "host-ai-session-events" }
    >
  >(
    [
      {
        scope: "host-ai-session-events",
        host_id: hostId,
        ai_session_id: session.id,
        page,
        per_page: EVENTS_PAGE_SIZE,
      },
    ],
    ({ queryKey }) => hostAPI.getHostAISessionEvents(queryKey[0]),
    { ...DEFAULT_USE_QUERY_OPTIONS, keepPreviousData: true }
  );

  const summary: ISummaryRow[] = [
    { id: "Agent", value: getAIAgentDisplay(session.agent).name },
    {
      id: "Started",
      value: formatTime(session.started_at, "MMM d, yyyy, pp"),
    },
    {
      id: "Duration",
      value: getAISessionDuration(session) ?? DEFAULT_EMPTY_CELL_VALUE,
    },
    {
      id: "User account",
      value: (
        <TooltipTruncatedText
          value={getAISessionUserAccount(session) || DEFAULT_EMPTY_CELL_VALUE}
        />
      ),
    },
    { id: "Model", value: session.model || DEFAULT_EMPTY_CELL_VALUE },
  ];

  const renderEvents = () => {
    if (isError) {
      return <DataError verticalPaddingSize="pad-medium" />;
    }
    if (isLoading || !data) {
      return <Spinner />;
    }
    if (!data.events.length && page === 0) {
      return session.backfill ? (
        <EmptyState
          header="Collecting events"
          info="Fleet is still collecting this session's events. Check back later."
          width="small"
          variant="list"
        />
      ) : (
        <EmptyState header="No events" width="small" variant="list" />
      );
    }
    return (
      <>
        {session.backfill && <InfoBanner>{AI_SESSION_BACKFILL_MESSAGE}</InfoBanner>}
        <ul className={`${baseClass}__events`}>
          {data.events.map((event) => {
            const isExpanded = expandedEventId === event.id;
            const payloadId = `${baseClass}__event-payload-${event.id}`;
            return (
              <li key={event.id} className={`${baseClass}__event`}>
                <button
                  type="button"
                  className={`${baseClass}__event-toggle`}
                  aria-expanded={isExpanded}
                  aria-controls={payloadId}
                  onClick={() =>
                    setExpandedEventId(isExpanded ? null : event.id)
                  }
                >
                  <Icon
                    name={isExpanded ? "chevron-down" : "chevron-right"}
                    color="core-fleet-black"
                  />
                  <span className={`${baseClass}__event-time`}>
                    {formatTime(event.executed_at, "pp")}
                  </span>
                  <span className={`${baseClass}__event-target`}>
                    <TooltipTruncatedText
                      value={event.target || DEFAULT_EMPTY_CELL_VALUE}
                    />
                  </span>
                </button>
                {isExpanded && (
                  <div className={`${baseClass}__event-details`}>
                    {event.truncated && (
                      <p className={`${baseClass}__event-truncated`}>
                        Truncated to 8 KB
                      </p>
                    )}
                    <pre
                      id={payloadId}
                      className={`${baseClass}__event-payload`}
                      // Scrollable, so it has to be focusable to scroll by keyboard.
                      // eslint-disable-next-line jsx-a11y/no-noninteractive-tabindex
                      tabIndex={0}
                      aria-label="Event payload"
                      // The payload comes from the host; syntaxHighlight escapes it.
                      dangerouslySetInnerHTML={{
                        __html: syntaxHighlight(event.payload),
                      }}
                    />
                  </div>
                )}
              </li>
            );
          })}
        </ul>
        <Pagination
          disablePrev={page === 0}
          disableNext={isPreviousData || !data.meta.has_next_results}
          hidePagination={page === 0 && !data.meta.has_next_results}
          onPrevPage={() => {
            setExpandedEventId(null);
            setPage(page - 1);
          }}
          onNextPage={() => {
            setExpandedEventId(null);
            setPage(page + 1);
          }}
        />
      </>
    );
  };

  return (
    <Modal
      className={baseClass}
      title={
        <span className={`${baseClass}__title`}>
          Session
          <span className={`${baseClass}__session-id`}>
            {session.agent_session_id}
          </span>
        </span>
      }
      width="large"
      onExit={onExit}
    >
      <div className={`${baseClass}__content`}>
        <dl className={`${baseClass}__summary`}>
          {summary.map((row) => (
            <div key={row.id} className={`${baseClass}__summary-row`}>
              <dt className={`${baseClass}__summary-label`}>{row.id}</dt>
              <dd className={`${baseClass}__summary-value`}>{row.value}</dd>
            </div>
          ))}
        </dl>
        <div className={`${baseClass}__events-section`}>
          <h3 className={`${baseClass}__events-title`}>
            Events
            {data && (
              <span className={`${baseClass}__events-count`}>
                {data.count} {stringUtils.pluralize(data.count, "event")}
              </span>
            )}
          </h3>
          {renderEvents()}
        </div>
      </div>
    </Modal>
  );
};

export default AISessionModal;
