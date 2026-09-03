import React, { useMemo } from "react";
import { useQuery } from "react-query";
import { format, parseISO } from "date-fns";

import chartsAPI, {
  IChartResponse,
  IChartApiParams,
  IChartQueryKey,
} from "services/entities/charts";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { DATASET_LABEL, IFormattedDataPoint } from "interfaces/charts";
import { HostPlatform, isIPadOrIPhone } from "interfaces/platform";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";

import CheckerboardViz from "pages/DashboardPage/cards/ChartCard/CheckerboardViz";
import DataCollectionDisabledState from "pages/DashboardPage/cards/ChartCard/DataCollectionDisabledState";

const baseClass = "host-online-history-modal";

const CHART_DAYS = 30;

const tooltipFormatter = ({ value }: { value: number }): string =>
  value === 1 ? "Online" : "Offline";

interface IHostOnlineHistoryModalProps {
  hostId: number;
  fleetId?: number;
  platform: HostPlatform;
  uptimeCollectionEnabled: boolean;
  onExit: () => void;
}

const HostOnlineHistoryModal = ({
  hostId,
  fleetId,
  platform,
  uptimeCollectionEnabled,
  onExit,
}: IHostOnlineHistoryModalProps): JSX.Element => {
  const queryParams: IChartApiParams = useMemo(
    () => ({
      // Extra day so the trailing 30 local days are always full regardless of TZ.
      days: CHART_DAYS + 1,
      tz_offset: new Date().getTimezoneOffset(),
      fleet_id: fleetId,
      include_host_ids: String(hostId),
    }),
    [hostId, fleetId]
  );

  const { data: chartData, isLoading, error } = useQuery<
    IChartResponse,
    Error,
    IChartResponse,
    IChartQueryKey[]
  >(
    [{ scope: "chart", metric: "uptime", params: queryParams }],
    () => chartsAPI.getChartData("uptime", queryParams),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: uptimeCollectionEnabled,
      staleTime: 300000, // 5 minutes
    }
  );

  const formattedData: IFormattedDataPoint[] = useMemo(() => {
    if (!chartData?.data) return [];
    // Single-host filter means each bucket's value is 0 or 1. Force percentage
    // to 0 or 100 so CheckerboardViz's fixed color ramp renders a clean
    // on/off grid.
    return chartData.data.map((point) => ({
      timestamp: point.timestamp,
      label: format(parseISO(point.timestamp), "MMM d, h:mm a"),
      value: point.value,
      percentage: point.value > 0 ? 100 : 0,
      total: 1,
    }));
  }, [chartData]);

  const renderChart = () => {
    if (!uptimeCollectionEnabled) {
      return (
        <DataCollectionDisabledState
          datasetLabel={DATASET_LABEL.uptime}
          currentTeamId={fleetId}
        />
      );
    }
    if (isLoading) {
      return <Spinner verticalPadding="small" />;
    }
    if (error) {
      return <DataError />;
    }
    if (!formattedData.length) {
      return (
        <div className={`${baseClass}__no-data`}>
          No chart data available yet.
        </div>
      );
    }
    return (
      <CheckerboardViz
        data={formattedData}
        selectedDays={CHART_DAYS}
        tooltipFormatter={tooltipFormatter}
        legendVariant="binary"
        legendInfo={
          isIPadOrIPhone(platform) ? (
            <TooltipWrapper
              tipContent="iOS/iPadOS hosts are online anytime they have power and an internet connection (including locked)."
              position="top"
              underline={false}
              showArrow
              tipOffset={8}
            >
              <Icon name="info-outline" />
            </TooltipWrapper>
          ) : undefined
        }
      />
    );
  };

  return (
    <Modal title="Online history" className={baseClass} onExit={onExit}>
      <>
        <div className={`${baseClass}__chart-container`}>{renderChart()}</div>
        <ModalFooter
          primaryButtons={
            <Button type="button" onClick={onExit}>
              Done
            </Button>
          }
        />
      </>
    </Modal>
  );
};

export default HostOnlineHistoryModal;
