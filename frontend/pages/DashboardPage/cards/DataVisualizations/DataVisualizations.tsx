import React from "react";
import { useQuery } from "react-query";
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  ArcElement,
  Title,
  Tooltip,
  Legend,
} from "chart.js";
import { Bar, Pie } from "react-chartjs-2";

import Card from "components/Card";
import LastUpdatedText from "components/LastUpdatedText";
import Spinner from "components/Spinner";
import DataError from "components/DataError";

import { IHostSummary } from "interfaces/host_summary";
import { ISoftwareResponse } from "interfaces/software";

import hostSummaryAPI from "services/entities/host_summary";
import softwareAPI, { ISoftwareApiParams } from "services/entities/software";
import {
  getOSVersions,
  IOSVersionsResponse,
} from "services/entities/operating_systems";

import useInfoCard from "../../components/InfoCard";
import "./_styles.scss";

// Register ChartJS components
ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  ArcElement,
  Title,
  Tooltip,
  Legend
);

interface IDataVisualizationsProps {
  teamId?: number;
  className?: string;
}

const DataVisualizations = ({
  teamId,
  className,
}: IDataVisualizationsProps): JSX.Element => {
  // Host summary data query
  const {
    data: hostSummaryData,
    isFetching: isHostSummaryFetching,
    error: errorHosts,
  } = useQuery<IHostSummary, Error, IHostSummary>(
    ["host summary", teamId],
    () => hostSummaryAPI.getSummary({ teamId }),
    {
      enabled: true,
      refetchInterval: 300000, // Refetch every 5 minutes
    }
  );

  // Software data query
  const {
    data: softwareData,
    isFetching: isSoftwareFetching,
    error: errorSoftware,
  } = useQuery<ISoftwareResponse, Error, ISoftwareResponse>(
    [
      {
        scope: "software",
        teamId,
        perPage: 10,
        orderKey: "hosts_count",
        orderDirection: "desc" as const,
      } as ISoftwareApiParams,
    ],
    ({ queryKey }) => softwareAPI.load(queryKey[0] as ISoftwareApiParams),
    {
      enabled: true,
      refetchInterval: 300000, // Refetch every 5 minutes
    }
  );

  // OS versions data query
  const {
    data: osVersionsData,
    isFetching: isOSVersionsFetching,
    error: errorOSVersions,
  } = useQuery<IOSVersionsResponse, Error, IOSVersionsResponse>(
    ["os_versions", teamId],
    () =>
      getOSVersions({
        teamId,
        per_page: 10,
        order_key: "hosts_count",
        order_direction: "desc",
      }),
    {
      enabled: true,
      refetchInterval: 300000, // Refetch every 5 minutes
    }
  );

  // Platform distribution chart data
  const platformChartData = React.useMemo(() => {
    if (!hostSummaryData?.platforms) {
      return {
        labels: [],
        datasets: [
          {
            data: [],
            backgroundColor: [],
          },
        ],
      };
    }

    const platformColors = {
      darwin: "rgba(54, 162, 235, 0.8)",
      windows: "rgba(75, 192, 192, 0.8)",
      linux: "rgba(255, 99, 132, 0.8)",
      chrome: "rgba(255, 206, 86, 0.8)",
      ios: "rgba(153, 102, 255, 0.8)",
      ipados: "rgba(255, 159, 64, 0.8)",
      android: "rgba(199, 199, 199, 0.8)",
    };

    const labels = hostSummaryData.platforms.map((p) => {
      // Convert platform names to more readable format
      const platformNames = {
        darwin: "macOS",
        windows: "Windows",
        linux: "Linux",
        chrome: "Chrome OS",
        ios: "iOS",
        ipados: "iPadOS",
        android: "Android",
      };
      return (
        platformNames[p.platform as keyof typeof platformNames] || p.platform
      );
    });

    const data = hostSummaryData.platforms.map((p) => p.hosts_count);
    const backgroundColor = hostSummaryData.platforms.map(
      (p) =>
        platformColors[p.platform as keyof typeof platformColors] ||
        "rgba(201, 203, 207, 0.8)"
    );

    return {
      labels,
      datasets: [
        {
          data,
          backgroundColor,
          borderColor: backgroundColor.map((color) =>
            color.replace("0.8", "1")
          ),
          borderWidth: 1,
        },
      ],
    };
  }, [hostSummaryData]);

  // Online/Offline hosts chart data
  const onlineStatusChartData = React.useMemo(() => {
    if (!hostSummaryData) {
      return {
        labels: [],
        datasets: [
          {
            data: [],
            backgroundColor: [],
          },
        ],
      };
    }

    return {
      labels: ["Online", "Offline", "Missing"],
      datasets: [
        {
          data: [
            hostSummaryData.online_count,
            hostSummaryData.offline_count,
            hostSummaryData.missing_30_days_count || 0,
          ],
          backgroundColor: [
            "rgba(75, 192, 92, 0.8)",
            "rgba(255, 99, 132, 0.8)",
            "rgba(255, 159, 64, 0.8)",
          ],
          borderColor: [
            "rgba(75, 192, 92, 1)",
            "rgba(255, 99, 132, 1)",
            "rgba(255, 159, 64, 1)",
          ],
          borderWidth: 1,
        },
      ],
    };
  }, [hostSummaryData]);

  // Top software chart data
  const topSoftwareChartData = React.useMemo(() => {
    if (!softwareData?.software || softwareData.software.length === 0) {
      return {
        labels: [],
        datasets: [
          {
            data: [],
            backgroundColor: [],
          },
        ],
      };
    }

    // Get top 10 software by host count
    const topSoftware = [...softwareData.software]
      .sort((a, b) => (b.hosts_count || 0) - (a.hosts_count || 0))
      .slice(0, 10);

    return {
      labels: topSoftware.map((s) => s.name),
      datasets: [
        {
          label: "Hosts Count",
          data: topSoftware.map((s) => s.hosts_count || 0),
          backgroundColor: "rgba(54, 162, 235, 0.8)",
          borderColor: "rgba(54, 162, 235, 1)",
          borderWidth: 1,
        },
      ],
    };
  }, [softwareData]);

  // OS versions chart data
  const osVersionsChartData = React.useMemo(() => {
    if (
      !osVersionsData?.os_versions ||
      osVersionsData.os_versions.length === 0
    ) {
      return {
        labels: [],
        datasets: [
          {
            data: [],
            backgroundColor: [],
          },
        ],
      };
    }

    // Group by OS name and version
    const osData = osVersionsData.os_versions.reduce((acc, os) => {
      const key = os.name_only;
      if (!acc[key]) {
        acc[key] = {
          name: key,
          hosts_count: 0,
          versions: {},
        };
      }

      acc[key].hosts_count += os.hosts_count;
      acc[key].versions[os.version] =
        (acc[key].versions[os.version] || 0) + os.hosts_count;

      return acc;
    }, {} as Record<string, { name: string; hosts_count: number; versions: Record<string, number> }>);

    // Convert to array and sort by host count
    const sortedOsData = Object.values(osData)
      .sort((a, b) => b.hosts_count - a.hosts_count)
      .slice(0, 5);

    // Generate colors for each OS
    const osColors = [
      "rgba(54, 162, 235, 0.8)",
      "rgba(75, 192, 192, 0.8)",
      "rgba(255, 99, 132, 0.8)",
      "rgba(255, 206, 86, 0.8)",
      "rgba(153, 102, 255, 0.8)",
    ];

    return {
      labels: sortedOsData.map((os) => os.name),
      datasets: [
        {
          label: "Hosts Count",
          data: sortedOsData.map((os) => os.hosts_count),
          backgroundColor: osColors,
          borderColor: osColors.map((color) => color.replace("0.8", "1")),
          borderWidth: 1,
        },
      ],
    };
  }, [osVersionsData]);

  // Vulnerabilities by OS chart data
  const vulnerabilitiesChartData = React.useMemo(() => {
    if (
      !osVersionsData?.os_versions ||
      osVersionsData.os_versions.length === 0
    ) {
      return {
        labels: [],
        datasets: [
          {
            data: [],
            backgroundColor: [],
          },
        ],
      };
    }

    // Count vulnerabilities by OS
    const vulnByOs = osVersionsData.os_versions.reduce((acc, os) => {
      if (os.vulnerabilities && os.vulnerabilities.length > 0) {
        acc[os.name_only] =
          (acc[os.name_only] || 0) + os.vulnerabilities.length;
      }
      return acc;
    }, {} as Record<string, number>);

    // Convert to arrays for chart
    const labels = Object.keys(vulnByOs);
    const data = Object.values(vulnByOs);

    // Generate colors
    const colors = [
      "rgba(255, 99, 132, 0.8)",
      "rgba(54, 162, 235, 0.8)",
      "rgba(255, 206, 86, 0.8)",
      "rgba(75, 192, 192, 0.8)",
      "rgba(153, 102, 255, 0.8)",
    ];

    return {
      labels,
      datasets: [
        {
          label: "Vulnerabilities Count",
          data,
          backgroundColor: colors.slice(0, labels.length),
          borderColor: colors
            .slice(0, labels.length)
            .map((color) => color.replace("0.8", "1")),
          borderWidth: 1,
        },
      ],
    };
  }, [osVersionsData]);

  const isLoading =
    isHostSummaryFetching || isSoftwareFetching || isOSVersionsFetching;
  const hasError = errorHosts || errorSoftware || errorOSVersions;

  // Create InfoCard components for each visualization
  const PlatformDistributionCard = useInfoCard({
    title: "Platform Distribution",
    children: (
      <div className="data-visualizations">
        {isLoading ? (
          <div className="data-visualizations__loading">
            <Spinner />
          </div>
        ) : (
          <>
            <div className="data-visualizations__chart">
              <Pie
                data={platformChartData}
                options={{
                  responsive: true,
                  plugins: {
                    legend: {
                      position: "right",
                    },
                    title: {
                      display: false,
                    },
                    tooltip: {
                      callbacks: {
                        label: (context) => {
                          const label = context.label || "";
                          const value = context.raw as number;
                          const total = (context.dataset
                            .data as number[]).reduce(
                            (a, b) => (a as number) + (b as number),
                            0
                          );
                          const percentage = Math.round((value / total) * 100);
                          return `${label}: ${value} (${percentage}%)`;
                        },
                      },
                    },
                  },
                }}
              />
            </div>
            <div className="data-visualizations__updated">
              <LastUpdatedText whatToRetrieve="platform data" />
            </div>
          </>
        )}
      </div>
    ),
  });

  const HostStatusCard = useInfoCard({
    title: "Host Status",
    children: (
      <div className="data-visualizations">
        {isLoading ? (
          <div className="data-visualizations__loading">
            <Spinner />
          </div>
        ) : (
          <>
            <div className="data-visualizations__chart">
              <Pie
                data={onlineStatusChartData}
                options={{
                  responsive: true,
                  plugins: {
                    legend: {
                      position: "right",
                    },
                    title: {
                      display: false,
                    },
                    tooltip: {
                      callbacks: {
                        label: (context) => {
                          const label = context.label || "";
                          const value = context.raw as number;
                          const total = (context.dataset
                            .data as number[]).reduce(
                            (a, b) => (a as number) + (b as number),
                            0
                          );
                          const percentage = Math.round((value / total) * 100);
                          return `${label}: ${value} (${percentage}%)`;
                        },
                      },
                    },
                  },
                }}
              />
            </div>
            <div className="data-visualizations__updated">
              <LastUpdatedText whatToRetrieve="host status" />
            </div>
          </>
        )}
      </div>
    ),
  });

  const TopSoftwareCard = useInfoCard({
    title: "Top Software by Host Count",
    children: (
      <div className="data-visualizations">
        {isLoading ? (
          <div className="data-visualizations__loading">
            <Spinner />
          </div>
        ) : (
          <>
            <div className="data-visualizations__chart">
              <Bar
                data={topSoftwareChartData}
                options={{
                  indexAxis: "y" as const,
                  responsive: true,
                  plugins: {
                    legend: {
                      display: false,
                    },
                    title: {
                      display: false,
                    },
                  },
                  scales: {
                    x: {
                      beginAtZero: true,
                    },
                  },
                }}
              />
            </div>
            {softwareData?.counts_updated_at && (
              <div className="data-visualizations__updated">
                <LastUpdatedText
                  lastUpdatedAt={softwareData.counts_updated_at}
                  whatToRetrieve="software data"
                />
              </div>
            )}
          </>
        )}
      </div>
    ),
  });

  const OSDistributionCard = useInfoCard({
    title: "Operating System Distribution",
    children: (
      <div className="data-visualizations">
        {isLoading ? (
          <div className="data-visualizations__loading">
            <Spinner />
          </div>
        ) : (
          <>
            <div className="data-visualizations__chart">
              <Pie
                data={osVersionsChartData}
                options={{
                  responsive: true,
                  plugins: {
                    legend: {
                      position: "right",
                    },
                    title: {
                      display: false,
                    },
                    tooltip: {
                      callbacks: {
                        label: (context) => {
                          const label = context.label || "";
                          const value = context.raw as number;
                          const total = (context.dataset
                            .data as number[]).reduce(
                            (a, b) => (a as number) + (b as number),
                            0
                          );
                          const percentage = Math.round((value / total) * 100);
                          return `${label}: ${value} (${percentage}%)`;
                        },
                      },
                    },
                  },
                }}
              />
            </div>
            {osVersionsData?.counts_updated_at && (
              <div className="data-visualizations__updated">
                <LastUpdatedText
                  lastUpdatedAt={osVersionsData.counts_updated_at}
                  whatToRetrieve="OS data"
                />
              </div>
            )}
          </>
        )}
      </div>
    ),
  });

  const VulnerabilitiesCard = useInfoCard({
    title: "Vulnerabilities by OS",
    children: (
      <div className="data-visualizations">
        {isLoading ? (
          <div className="data-visualizations__loading">
            <Spinner />
          </div>
        ) : (
          <>
            <div className="data-visualizations__chart">
              <Bar
                data={vulnerabilitiesChartData}
                options={{
                  responsive: true,
                  plugins: {
                    legend: {
                      display: false,
                    },
                    title: {
                      display: false,
                    },
                  },
                  scales: {
                    y: {
                      beginAtZero: true,
                    },
                  },
                }}
              />
            </div>
            {osVersionsData?.counts_updated_at && (
              <div className="data-visualizations__updated">
                <LastUpdatedText
                  lastUpdatedAt={osVersionsData.counts_updated_at}
                  whatToRetrieve="vulnerability data"
                />
              </div>
            )}
          </>
        )}
      </div>
    ),
  });

  if (hasError) {
    return (
      <Card className={className}>
        <DataError />
      </Card>
    );
  }

  return (
    <>
      {PlatformDistributionCard}
      {HostStatusCard}
      {TopSoftwareCard}
      {OSDistributionCard}
      {VulnerabilitiesCard}
    </>
  );
};

export default DataVisualizations;
