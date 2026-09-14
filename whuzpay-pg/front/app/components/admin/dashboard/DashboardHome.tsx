import EarningsOverview from "./EarningsOverview";
import MetricCards from "./MetricCards";
import OrderStatusTable from "./OrderStatusTable";
import RecentActivities from "./RecentActivities";
import SalesChart from "./SalesChart";
import StatusCards from "./StatusCards";
import TrafficChart from "./TrafficChart";

export default function DashboardHome() {
  return (
    <div className="flex w-full flex-col gap-4 lg:gap-4.5">
      {/* Top row: earnings + sales chart + traffic */}
      <div className="grid grid-cols-1 items-stretch gap-4 lg:grid-cols-12 lg:gap-4.5">
        <div className="lg:col-span-3 lg:min-h-75">
          <EarningsOverview />
        </div>
        <div className="lg:col-span-6 lg:min-h-75">
          <SalesChart />
        </div>
        <div className="lg:col-span-3 lg:min-h-75">
          <TrafficChart />
        </div>
      </div>

      {/* Metric cards */}
      <MetricCards />

      {/* Colored status cards */}
      <StatusCards />

      {/* Bottom: activities + orders */}
      <div className="grid grid-cols-1 items-stretch gap-4 lg:grid-cols-12 lg:gap-4.5">
        <div className="lg:col-span-4">
          <RecentActivities />
        </div>
        <div className="lg:col-span-8">
          <OrderStatusTable />
        </div>
      </div>
    </div>
  );
}
